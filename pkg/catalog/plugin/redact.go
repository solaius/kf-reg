package plugin

import "strings"

// RedactedValue is the replacement string for sensitive property values.
const RedactedValue = "***REDACTED***"

// sensitiveKeyPatterns are property key substrings that indicate sensitive values.
var sensitiveKeyPatterns = []string{"password", "token", "secret", "apikey", "api_key", "credential"}

// IsSensitiveKey checks if a property key indicates a sensitive value.
// The check is case-insensitive and matches any of the known sensitive patterns.
func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, p := range sensitiveKeyPatterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// RedactSensitiveProperties returns a copy of properties with sensitive values
// replaced by RedactedValue. A property is considered sensitive if its key
// (case-insensitive) contains any of the sensitive patterns AND its value is a
// plain string (not a map, which would indicate a SecretRef object).
func RedactSensitiveProperties(props map[string]any) map[string]any {
	if props == nil {
		return nil
	}
	out := make(map[string]any, len(props))
	for k, v := range props {
		if IsSensitiveKey(k) {
			// Only redact plain string values. Maps (SecretRef-like objects)
			// are passed through unchanged.
			if _, isMap := v.(map[string]any); !isMap {
				out[k] = RedactedValue
				continue
			}
		}
		out[k] = v
	}
	return out
}
