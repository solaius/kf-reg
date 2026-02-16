package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sSecretResolver_Resolve(t *testing.T) {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("super-secret-value"),
			"token":   []byte("bearer-token-123"),
		},
	}
	client := fake.NewSimpleClientset(secret)
	resolver := NewK8sSecretResolver(client, "default")

	t.Run("resolves existing key", func(t *testing.T) {
		val, err := resolver.Resolve(ctx, SecretRef{Name: "my-secret", Key: "api-key"})
		require.NoError(t, err)
		assert.Equal(t, "super-secret-value", val)
	})

	t.Run("resolves second key from same secret", func(t *testing.T) {
		val, err := resolver.Resolve(ctx, SecretRef{Name: "my-secret", Key: "token"})
		require.NoError(t, err)
		assert.Equal(t, "bearer-token-123", val)
	})

	t.Run("uses explicit namespace", func(t *testing.T) {
		nsSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ns-secret",
				Namespace: "other-ns",
			},
			Data: map[string][]byte{
				"value": []byte("ns-value"),
			},
		}
		nsClient := fake.NewSimpleClientset(nsSecret)
		nsResolver := NewK8sSecretResolver(nsClient, "default")

		val, err := nsResolver.Resolve(ctx, SecretRef{Name: "ns-secret", Namespace: "other-ns", Key: "value"})
		require.NoError(t, err)
		assert.Equal(t, "ns-value", val)
	})

	t.Run("missing secret returns error", func(t *testing.T) {
		_, err := resolver.Resolve(ctx, SecretRef{Name: "nonexistent", Key: "key"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get Secret")
	})

	t.Run("missing key returns error", func(t *testing.T) {
		_, err := resolver.Resolve(ctx, SecretRef{Name: "my-secret", Key: "nonexistent-key"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "key \"nonexistent-key\" not found")
	})

	t.Run("uses default namespace when ref has no namespace", func(t *testing.T) {
		val, err := resolver.Resolve(ctx, SecretRef{Name: "my-secret", Key: "api-key"})
		require.NoError(t, err)
		assert.Equal(t, "super-secret-value", val)
	})

	t.Run("defaults to custom namespace when ref namespace is empty", func(t *testing.T) {
		// Create a secret in a custom namespace to verify that defaultNamespace
		// is used (not hard-coded "default") when ref.Namespace is empty.
		customNS := "my-app-namespace"
		customSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom-secret",
				Namespace: customNS,
			},
			Data: map[string][]byte{
				"password": []byte("custom-ns-password"),
			},
		}
		customClient := fake.NewSimpleClientset(customSecret)
		customResolver := NewK8sSecretResolver(customClient, customNS)

		// Resolve without specifying namespace -- should use customNS.
		val, err := customResolver.Resolve(ctx, SecretRef{Name: "custom-secret", Key: "password"})
		require.NoError(t, err)
		assert.Equal(t, "custom-ns-password", val)
	})

	t.Run("explicit namespace overrides default namespace", func(t *testing.T) {
		// Secret exists in "explicit-ns", but resolver default is "wrong-ns".
		// Providing Namespace in the ref should use the explicit value.
		explicitSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "explicit-secret",
				Namespace: "explicit-ns",
			},
			Data: map[string][]byte{
				"key": []byte("explicit-value"),
			},
		}
		explicitClient := fake.NewSimpleClientset(explicitSecret)
		explicitResolver := NewK8sSecretResolver(explicitClient, "wrong-ns")

		val, err := explicitResolver.Resolve(ctx, SecretRef{
			Name:      "explicit-secret",
			Namespace: "explicit-ns",
			Key:       "key",
		})
		require.NoError(t, err)
		assert.Equal(t, "explicit-value", val)

		// Without explicit namespace, it should fail because default is "wrong-ns"
		// and the secret is in "explicit-ns".
		_, err = explicitResolver.Resolve(ctx, SecretRef{
			Name: "explicit-secret",
			Key:  "key",
		})
		require.Error(t, err, "should fail when defaulting to wrong-ns")
	})
}

func TestIsSecretRef(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantRef SecretRef
		wantOK  bool
	}{
		{
			name:    "valid SecretRef with name and key",
			value:   map[string]any{"name": "my-secret", "key": "api-key"},
			wantRef: SecretRef{Name: "my-secret", Key: "api-key"},
			wantOK:  true,
		},
		{
			name:    "valid SecretRef with namespace",
			value:   map[string]any{"name": "my-secret", "namespace": "prod", "key": "token"},
			wantRef: SecretRef{Name: "my-secret", Namespace: "prod", Key: "token"},
			wantOK:  true,
		},
		{
			name:   "missing name",
			value:  map[string]any{"key": "api-key"},
			wantOK: false,
		},
		{
			name:   "missing key",
			value:  map[string]any{"name": "my-secret"},
			wantOK: false,
		},
		{
			name:   "empty name",
			value:  map[string]any{"name": "", "key": "api-key"},
			wantOK: false,
		},
		{
			name:   "empty key",
			value:  map[string]any{"name": "my-secret", "key": ""},
			wantOK: false,
		},
		{
			name:   "not a map",
			value:  "plain-string",
			wantOK: false,
		},
		{
			name:   "nil value",
			value:  nil,
			wantOK: false,
		},
		{
			name:   "name is not string",
			value:  map[string]any{"name": 123, "key": "api-key"},
			wantOK: false,
		},
		{
			name:   "key is not string",
			value:  map[string]any{"name": "my-secret", "key": 456},
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, ok := IsSecretRef(tc.value)
			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, tc.wantRef, ref)
			}
		})
	}
}

func TestResolveSecretRefs(t *testing.T) {
	ctx := context.Background()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("resolved-api-key"),
			"token":   []byte("resolved-token"),
		},
	}
	client := fake.NewSimpleClientset(secret)
	resolver := NewK8sSecretResolver(client, "default")

	t.Run("nil resolver returns props unchanged", func(t *testing.T) {
		props := map[string]any{"url": "https://example.com"}
		result, err := ResolveSecretRefs(ctx, props, nil)
		require.NoError(t, err)
		assert.Equal(t, props, result)
	})

	t.Run("nil props returns nil", func(t *testing.T) {
		result, err := ResolveSecretRefs(ctx, nil, resolver)
		require.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("no secret refs returns copy", func(t *testing.T) {
		props := map[string]any{
			"url":  "https://example.com",
			"name": "test",
		}
		result, err := ResolveSecretRefs(ctx, props, resolver)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", result["url"])
		assert.Equal(t, "test", result["name"])
	})

	t.Run("resolves SecretRef values", func(t *testing.T) {
		props := map[string]any{
			"url":    "https://example.com",
			"apiKey": map[string]any{"name": "my-secret", "key": "api-key"},
		}
		result, err := ResolveSecretRefs(ctx, props, resolver)
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", result["url"])
		assert.Equal(t, "resolved-api-key", result["apiKey"])
	})

	t.Run("resolves multiple SecretRefs", func(t *testing.T) {
		props := map[string]any{
			"apiKey": map[string]any{"name": "my-secret", "key": "api-key"},
			"token":  map[string]any{"name": "my-secret", "key": "token"},
			"url":    "https://example.com",
		}
		result, err := ResolveSecretRefs(ctx, props, resolver)
		require.NoError(t, err)
		assert.Equal(t, "resolved-api-key", result["apiKey"])
		assert.Equal(t, "resolved-token", result["token"])
		assert.Equal(t, "https://example.com", result["url"])
	})

	t.Run("does not mutate original map", func(t *testing.T) {
		secretRefMap := map[string]any{"name": "my-secret", "key": "api-key"}
		props := map[string]any{
			"apiKey": secretRefMap,
		}
		_, err := ResolveSecretRefs(ctx, props, resolver)
		require.NoError(t, err)
		// Original map should still have the SecretRef map
		assert.Equal(t, secretRefMap, props["apiKey"])
	})

	t.Run("returns error for missing secret", func(t *testing.T) {
		props := map[string]any{
			"apiKey": map[string]any{"name": "nonexistent", "key": "key"},
		}
		_, err := ResolveSecretRefs(ctx, props, resolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve secret")
	})

	t.Run("returns error for missing key", func(t *testing.T) {
		props := map[string]any{
			"apiKey": map[string]any{"name": "my-secret", "key": "nonexistent"},
		}
		_, err := ResolveSecretRefs(ctx, props, resolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve secret")
	})

	t.Run("maps without name and key are not treated as SecretRef", func(t *testing.T) {
		arbitraryMap := map[string]any{"foo": "bar", "baz": 42}
		props := map[string]any{
			"config": arbitraryMap,
		}
		result, err := ResolveSecretRefs(ctx, props, resolver)
		require.NoError(t, err)
		assert.Equal(t, arbitraryMap, result["config"])
	})
}

// TestSecretRefResolution_E2E_FullFlow exercises the complete lifecycle of
// SecretRef handling: K8s Secret creation -> resolve -> verify value ->
// verify original is unmutated -> verify redaction on resolved output.
// This proves the integration between K8sSecretResolver, ResolveSecretRefs,
// and RedactSensitiveProperties works end-to-end.
func TestSecretRefResolution_E2E_FullFlow(t *testing.T) {
	ctx := context.Background()

	// 1. Create fake K8s Secrets in two namespaces.
	defaultSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-credentials",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"api-key": []byte("sk-live-abc123"),
			"token":   []byte("bearer-xyz789"),
		},
	}
	prodSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-credentials",
			Namespace: "production",
		},
		Data: map[string][]byte{
			"password": []byte("p@ssw0rd!"),
		},
	}
	k8sClient := fake.NewSimpleClientset(defaultSecret, prodSecret)
	resolver := NewK8sSecretResolver(k8sClient, "default")

	// 2. Build source config properties mixing plain values, SecretRef from
	//    default namespace, and SecretRef from explicit namespace.
	originalProps := map[string]any{
		"url": "https://models.example.com/catalog",
		"apiKey": map[string]any{
			"name": "api-credentials",
			"key":  "api-key",
		},
		"token": map[string]any{
			"name": "api-credentials",
			"key":  "token",
		},
		"password": map[string]any{
			"name":      "prod-credentials",
			"namespace": "production",
			"key":       "password",
		},
		"maxRetries": 3,
	}

	// Deep-copy the original SecretRef maps so we can verify non-mutation later.
	origAPIKeyRef := map[string]any{"name": "api-credentials", "key": "api-key"}
	origTokenRef := map[string]any{"name": "api-credentials", "key": "token"}
	origPasswordRef := map[string]any{"name": "prod-credentials", "namespace": "production", "key": "password"}

	// 3. Resolve SecretRefs.
	resolved, err := ResolveSecretRefs(ctx, originalProps, resolver)
	require.NoError(t, err)

	// 4. Verify all SecretRefs were resolved to their actual values.
	assert.Equal(t, "sk-live-abc123", resolved["apiKey"], "apiKey should resolve to secret value")
	assert.Equal(t, "bearer-xyz789", resolved["token"], "token should resolve to secret value")
	assert.Equal(t, "p@ssw0rd!", resolved["password"], "password should resolve from explicit namespace")

	// 5. Verify plain values passed through unchanged.
	assert.Equal(t, "https://models.example.com/catalog", resolved["url"])
	assert.Equal(t, 3, resolved["maxRetries"])

	// 6. Verify the original properties map was NOT mutated.
	apiKeyVal, ok := originalProps["apiKey"].(map[string]any)
	require.True(t, ok, "original apiKey should still be a map, got %T", originalProps["apiKey"])
	assert.Equal(t, origAPIKeyRef, apiKeyVal)

	tokenVal, ok := originalProps["token"].(map[string]any)
	require.True(t, ok, "original token should still be a map, got %T", originalProps["token"])
	assert.Equal(t, origTokenRef, tokenVal)

	passwordVal, ok := originalProps["password"].(map[string]any)
	require.True(t, ok, "original password should still be a map, got %T", originalProps["password"])
	assert.Equal(t, origPasswordRef, passwordVal)

	// 7. Verify redaction works on the resolved output.
	// After resolution, "apiKey" contains plain string "sk-live-abc123",
	// "token" contains "bearer-xyz789", "password" contains "p@ssw0rd!".
	// RedactSensitiveProperties should redact keys matching sensitive patterns.
	redacted := RedactSensitiveProperties(resolved)

	// "token" matches the "token" pattern -> should be redacted.
	assert.Equal(t, RedactedValue, redacted["token"], "resolved token should be redacted")
	// "password" matches the "password" pattern -> should be redacted.
	assert.Equal(t, RedactedValue, redacted["password"], "resolved password should be redacted")
	// "apiKey" matches "apikey" pattern (case-insensitive) -> should be redacted.
	assert.Equal(t, RedactedValue, redacted["apiKey"], "resolved apiKey should be redacted")
	// "url" is not sensitive -> should pass through.
	assert.Equal(t, "https://models.example.com/catalog", redacted["url"])
	// "maxRetries" is not sensitive -> should pass through.
	assert.Equal(t, 3, redacted["maxRetries"])

	// 8. Verify redaction of the ORIGINAL (unresolved) properties.
	// SecretRef maps should NOT be redacted (redaction only applies to plain strings).
	redactedOriginal := RedactSensitiveProperties(originalProps)
	_, isMap := redactedOriginal["apiKey"].(map[string]any)
	assert.True(t, isMap, "unresolved apiKey (map) should not be redacted")
	_, isMap = redactedOriginal["token"].(map[string]any)
	assert.True(t, isMap, "unresolved token (map) should not be redacted")
	_, isMap = redactedOriginal["password"].(map[string]any)
	assert.True(t, isMap, "unresolved password (map) should not be redacted")

	// 9. Simulate a provider callback receiving the resolved properties.
	providerCalled := false
	providerFn := func(props map[string]any) error {
		providerCalled = true
		assert.Equal(t, "sk-live-abc123", props["apiKey"])
		assert.Equal(t, "bearer-xyz789", props["token"])
		assert.Equal(t, "p@ssw0rd!", props["password"])
		assert.Equal(t, "https://models.example.com/catalog", props["url"])
		return nil
	}
	err = providerFn(resolved)
	require.NoError(t, err)
	assert.True(t, providerCalled, "provider function should have been called")
}
