package plugin

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTRoleExtractor(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate RSA key")

	createToken := func(claims jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, err := token.SignedString(privateKey)
		require.NoError(t, err, "failed to sign token")
		return tokenString
	}

	tests := []struct {
		name     string
		token    string
		config   JWTRoleExtractorConfig
		expected Role
	}{
		{
			name:     "no authorization header",
			token:    "",
			config:   JWTRoleExtractorConfig{},
			expected: RoleViewer,
		},
		{
			name:  "operator role from simple claim",
			token: createToken(jwt.MapClaims{"role": "operator", "exp": time.Now().Add(time.Hour).Unix()}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "role",
				OperatorRoleValue: "operator",
			},
			expected: RoleOperator,
		},
		{
			name:  "viewer role from simple claim",
			token: createToken(jwt.MapClaims{"role": "viewer", "exp": time.Now().Add(time.Hour).Unix()}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "role",
				OperatorRoleValue: "operator",
			},
			expected: RoleViewer,
		},
		{
			name: "operator from nested claim (Keycloak-style)",
			token: createToken(jwt.MapClaims{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"user", "operator"},
				},
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "realm_access.roles",
				OperatorRoleValue: "operator",
			},
			expected: RoleOperator,
		},
		{
			name: "viewer when operator not in array",
			token: createToken(jwt.MapClaims{
				"realm_access": map[string]interface{}{
					"roles": []interface{}{"user", "read-only"},
				},
				"exp": time.Now().Add(time.Hour).Unix(),
			}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "realm_access.roles",
				OperatorRoleValue: "operator",
			},
			expected: RoleViewer,
		},
		{
			name:  "missing claim defaults to viewer",
			token: createToken(jwt.MapClaims{"sub": "user1", "exp": time.Now().Add(time.Hour).Unix()}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "role",
				OperatorRoleValue: "operator",
			},
			expected: RoleViewer,
		},
		{
			name:     "malformed token defaults to viewer",
			token:    "not.a.valid.jwt",
			config:   JWTRoleExtractorConfig{},
			expected: RoleViewer,
		},
		{
			name:  "case-insensitive role matching",
			token: createToken(jwt.MapClaims{"role": "Operator", "exp": time.Now().Add(time.Hour).Unix()}),
			config: JWTRoleExtractorConfig{
				RoleClaim:         "role",
				OperatorRoleValue: "operator",
			},
			expected: RoleOperator,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use ParseUnverified mode for tests (no public key)
			extractor, err := NewJWTRoleExtractor(tt.config)
			require.NoError(t, err, "failed to create extractor")

			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			got := extractor(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{"empty header", "", ""},
		{"bearer token", "Bearer abc123", "abc123"},
		{"bearer lowercase", "bearer abc123", "abc123"},
		{"no bearer prefix", "Basic abc123", ""},
		{"bearer with extra spaces", "Bearer  abc123 ", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := extractBearerToken(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractRoleFromClaims(t *testing.T) {
	tests := []struct {
		name          string
		claims        jwt.MapClaims
		claimPath     string
		operatorValue string
		expected      Role
	}{
		{
			name:          "simple string match",
			claims:        jwt.MapClaims{"role": "operator"},
			claimPath:     "role",
			operatorValue: "operator",
			expected:      RoleOperator,
		},
		{
			name:          "nested claim",
			claims:        jwt.MapClaims{"app": map[string]interface{}{"role": "operator"}},
			claimPath:     "app.role",
			operatorValue: "operator",
			expected:      RoleOperator,
		},
		{
			name:          "array with operator",
			claims:        jwt.MapClaims{"roles": []interface{}{"admin", "operator"}},
			claimPath:     "roles",
			operatorValue: "operator",
			expected:      RoleOperator,
		},
		{
			name:          "missing path",
			claims:        jwt.MapClaims{},
			claimPath:     "role",
			operatorValue: "operator",
			expected:      RoleViewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRoleFromClaims(tt.claims, tt.claimPath, tt.operatorValue)
			assert.Equal(t, tt.expected, got)
		})
	}
}
