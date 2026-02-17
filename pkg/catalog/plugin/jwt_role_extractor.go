package plugin

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// JWTRoleExtractorConfig configures the JWT-based role extractor.
type JWTRoleExtractorConfig struct {
	// RoleClaim is the JWT claim path containing the user's role.
	// Supports dot-notation for nested claims (e.g., "realm_access.roles").
	// Default: "role"
	RoleClaim string

	// OperatorRoleValue is the claim value that maps to RoleOperator.
	// Any other value (or missing claim) maps to RoleViewer.
	// Default: "operator"
	OperatorRoleValue string

	// PublicKeyPath is the path to the PEM-encoded RSA public key for RS256 verification.
	// If empty, tokens are parsed but NOT verified (suitable for dev/testing with trusted proxies).
	PublicKeyPath string

	// Issuer is the expected token issuer (iss claim). If empty, issuer is not validated.
	Issuer string

	// Audience is the expected token audience (aud claim). If empty, audience is not validated.
	Audience string

	// Logger for debugging. If nil, uses slog.Default().
	Logger *slog.Logger
}

// NewJWTRoleExtractor creates a RoleExtractor that reads roles from JWT Bearer tokens.
// The token is expected in the Authorization header: "Bearer <token>".
//
// Security model:
//   - If PublicKeyPath is set, tokens are cryptographically verified (RS256)
//   - If PublicKeyPath is empty, tokens are parsed without verification (trusted proxy mode)
//   - Missing or invalid tokens default to RoleViewer (deny by default for operator access)
func NewJWTRoleExtractor(cfg JWTRoleExtractorConfig) (RoleExtractor, error) {
	if cfg.RoleClaim == "" {
		cfg.RoleClaim = "role"
	}
	if cfg.OperatorRoleValue == "" {
		cfg.OperatorRoleValue = "operator"
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	var publicKey *rsa.PublicKey
	if cfg.PublicKeyPath != "" {
		keyData, err := os.ReadFile(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read JWT public key from %s: %w", cfg.PublicKeyPath, err)
		}
		block, _ := pem.Decode(keyData)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block from %s", cfg.PublicKeyPath)
		}
		parsedKey, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key: %w", err)
		}
		rsaKey, ok := parsedKey.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA (got %T)", parsedKey)
		}
		publicKey = rsaKey
		cfg.Logger.Info("JWT role extractor: using RS256 verification", "keyPath", cfg.PublicKeyPath)
	} else {
		cfg.Logger.Warn("JWT role extractor: no public key configured, tokens parsed without verification (trusted proxy mode)")
	}

	return func(r *http.Request) Role {
		token := extractBearerToken(r)
		if token == "" {
			return RoleViewer
		}

		claims, err := parseJWTClaims(token, publicKey, cfg)
		if err != nil {
			cfg.Logger.Debug("JWT parse failed, defaulting to viewer", "error", err)
			return RoleViewer
		}

		role := extractRoleFromClaims(claims, cfg.RoleClaim, cfg.OperatorRoleValue)
		return role
	}, nil
}

// extractBearerToken extracts the token from "Authorization: Bearer <token>".
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// parseJWTClaims parses and optionally verifies a JWT token.
func parseJWTClaims(tokenString string, publicKey *rsa.PublicKey, cfg JWTRoleExtractorConfig) (jwt.MapClaims, error) {
	parserOpts := []jwt.ParserOption{}
	if cfg.Issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(cfg.Issuer))
	}
	if cfg.Audience != "" {
		parserOpts = append(parserOpts, jwt.WithAudience(cfg.Audience))
	}

	var token *jwt.Token
	var err error

	if publicKey != nil {
		token, err = jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return publicKey, nil
		}, parserOpts...)
	} else {
		// Trusted proxy mode: parse without verification
		parser := jwt.NewParser(parserOpts...)
		token, _, err = parser.ParseUnverified(tokenString, jwt.MapClaims{})
	}

	if err != nil {
		return nil, fmt.Errorf("JWT parse error: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unexpected claims type")
	}

	return claims, nil
}

// extractRoleFromClaims extracts the role from JWT claims.
// Supports dot-notation for nested claims (e.g., "realm_access.roles").
// For array claims, checks if operatorValue is present in the array.
func extractRoleFromClaims(claims jwt.MapClaims, claimPath string, operatorValue string) Role {
	parts := strings.Split(claimPath, ".")
	var current interface{} = map[string]interface{}(claims)

	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return RoleViewer
		}
		current, ok = m[part]
		if !ok {
			return RoleViewer
		}
	}

	// Handle string claim
	if strVal, ok := current.(string); ok {
		if strings.EqualFold(strVal, operatorValue) {
			return RoleOperator
		}
		return RoleViewer
	}

	// Handle array claim (e.g., Keycloak realm_access.roles: ["operator", "user"])
	if arrVal, ok := current.([]interface{}); ok {
		for _, v := range arrVal {
			if s, ok := v.(string); ok && strings.EqualFold(s, operatorValue) {
				return RoleOperator
			}
		}
	}

	return RoleViewer
}

// WithJWTRoleExtractor is a convenience ServerOption that configures JWT-based auth.
func WithJWTRoleExtractor(cfg JWTRoleExtractorConfig) ServerOption {
	return func(s *Server) {
		extractor, err := NewJWTRoleExtractor(cfg)
		if err != nil {
			if cfg.Logger != nil {
				cfg.Logger.Error("failed to create JWT role extractor", "error", err)
			}
			return
		}
		s.roleExtractor = extractor
	}
}
