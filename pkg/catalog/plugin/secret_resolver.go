package plugin

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SecretResolver resolves SecretRef objects into their actual string values.
type SecretResolver interface {
	Resolve(ctx context.Context, ref SecretRef) (string, error)
}

// K8sSecretResolver reads secret values from Kubernetes Secrets.
type K8sSecretResolver struct {
	client           kubernetes.Interface
	defaultNamespace string
}

// NewK8sSecretResolver creates a SecretResolver backed by the Kubernetes API.
// The defaultNamespace is used when a SecretRef does not specify a namespace.
func NewK8sSecretResolver(client kubernetes.Interface, defaultNamespace string) *K8sSecretResolver {
	return &K8sSecretResolver{
		client:           client,
		defaultNamespace: defaultNamespace,
	}
}

// Resolve reads the referenced key from the Kubernetes Secret.
// If ref.Namespace is empty, the resolver's defaultNamespace is used. This
// allows source configs to omit the namespace field and have the server
// automatically look up secrets in its own namespace (set at startup via
// CATALOG_CONFIG_NAMESPACE).
func (r *K8sSecretResolver) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	ns := ref.Namespace
	if ns == "" {
		ns = r.defaultNamespace
	}

	secret, err := r.client.CoreV1().Secrets(ns).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Secret %s/%s: %w", ns, ref.Name, err)
	}

	data, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in Secret %s/%s", ref.Key, ns, ref.Name)
	}

	return string(data), nil
}

// IsSecretRef checks whether a property value looks like a SecretRef object.
// A SecretRef-shaped map has string "name" and "key" fields.
func IsSecretRef(v any) (SecretRef, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return SecretRef{}, false
	}

	name, nameOK := m["name"].(string)
	key, keyOK := m["key"].(string)
	if !nameOK || !keyOK || name == "" || key == "" {
		return SecretRef{}, false
	}

	ref := SecretRef{
		Name: name,
		Key:  key,
	}
	if ns, ok := m["namespace"].(string); ok {
		ref.Namespace = ns
	}

	return ref, true
}

// ResolveSecretRefs walks a properties map and replaces SecretRef-shaped values
// with the resolved secret data. It returns a shallow copy of the map with
// resolved values; the original map is not modified.
// If resolver is nil, the original properties are returned unchanged.
func ResolveSecretRefs(ctx context.Context, props map[string]any, resolver SecretResolver) (map[string]any, error) {
	if resolver == nil || props == nil {
		return props, nil
	}

	out := make(map[string]any, len(props))
	for k, v := range props {
		ref, ok := IsSecretRef(v)
		if !ok {
			out[k] = v
			continue
		}

		resolved, err := resolver.Resolve(ctx, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve secret for property %q: %w", k, err)
		}
		out[k] = resolved
	}

	return out, nil
}
