package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"Password", true},
		{"PASSWORD", true},
		{"dbPassword", true},
		{"token", true},
		{"authToken", true},
		{"AUTH_TOKEN", true},
		{"secret", true},
		{"clientSecret", true},
		{"apikey", true},
		{"apiKey", true},
		{"APIKEY", true},
		{"api_key", true},
		{"API_KEY", true},
		{"credential", true},
		{"credentials", true},
		{"username", false},
		{"url", false},
		{"path", false},
		{"name", false},
		{"enabled", false},
		{"content", false},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			assert.Equal(t, tc.want, IsSensitiveKey(tc.key))
		})
	}
}

func TestRedactSensitiveProperties(t *testing.T) {
	t.Run("nil properties returns nil", func(t *testing.T) {
		assert.Nil(t, RedactSensitiveProperties(nil))
	})

	t.Run("empty properties returns empty", func(t *testing.T) {
		result := RedactSensitiveProperties(map[string]any{})
		assert.Empty(t, result)
	})

	t.Run("non-sensitive properties are unchanged", func(t *testing.T) {
		props := map[string]any{
			"url":     "https://example.com",
			"name":    "my-source",
			"enabled": true,
			"count":   42,
		}
		result := RedactSensitiveProperties(props)
		assert.Equal(t, "https://example.com", result["url"])
		assert.Equal(t, "my-source", result["name"])
		assert.Equal(t, true, result["enabled"])
		assert.Equal(t, 42, result["count"])
	})

	t.Run("sensitive string values are redacted", func(t *testing.T) {
		props := map[string]any{
			"url":    "https://example.com",
			"apiKey": "sk-secret-key-12345",
			"token":  "bearer-token-xyz",
		}
		result := RedactSensitiveProperties(props)
		assert.Equal(t, "https://example.com", result["url"])
		assert.Equal(t, RedactedValue, result["apiKey"])
		assert.Equal(t, RedactedValue, result["token"])
	})

	t.Run("SecretRef map values are NOT redacted", func(t *testing.T) {
		secretRef := map[string]any{
			"name":      "my-secret",
			"namespace": "default",
			"key":       "api-key",
		}
		props := map[string]any{
			"url":       "https://example.com",
			"apiKeyRef": secretRef,
		}
		result := RedactSensitiveProperties(props)
		assert.Equal(t, "https://example.com", result["url"])
		// apiKeyRef contains "apikey" pattern but value is a map, so not redacted.
		assert.Equal(t, secretRef, result["apiKeyRef"])
	})

	t.Run("mixed sensitive and non-sensitive", func(t *testing.T) {
		props := map[string]any{
			"url":          "https://hf.co",
			"password":     "hunter2",
			"credential":   "my-cred",
			"path":         "/data",
			"clientSecret": "s3cr3t",
		}
		result := RedactSensitiveProperties(props)
		assert.Equal(t, "https://hf.co", result["url"])
		assert.Equal(t, RedactedValue, result["password"])
		assert.Equal(t, RedactedValue, result["credential"])
		assert.Equal(t, "/data", result["path"])
		assert.Equal(t, RedactedValue, result["clientSecret"])
	})

	t.Run("original properties are not modified", func(t *testing.T) {
		props := map[string]any{
			"apiKey": "original-value",
		}
		_ = RedactSensitiveProperties(props)
		assert.Equal(t, "original-value", props["apiKey"])
	})

	t.Run("non-string sensitive values are redacted", func(t *testing.T) {
		props := map[string]any{
			"token": 12345, // numeric, not a map -> should be redacted
		}
		result := RedactSensitiveProperties(props)
		assert.Equal(t, RedactedValue, result["token"])
	})
}
