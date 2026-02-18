package k8mocks

import (
	"context"
	"fmt"
	"log/slog"

	k8s "github.com/kubeflow/model-registry/ui/bff/internal/integrations/kubernetes"
	"k8s.io/client-go/kubernetes"
)

type InternalKubernetesClientMock struct {
	*k8s.InternalKubernetesClient
}

// newMockedInternalKubernetesClientFromClientset creates a mock from existing envtest clientset
func newMockedInternalKubernetesClientFromClientset(clientset kubernetes.Interface, logger *slog.Logger) k8s.KubernetesClientInterface {
	return &InternalKubernetesClientMock{
		InternalKubernetesClient: &k8s.InternalKubernetesClient{
			SharedClientLogic: k8s.SharedClientLogic{
				Client: clientset,
				Logger: logger,
			},
		},
	}
}

// GetServiceDetails overrides to simulate ClusterIP for localhost access
func (m *InternalKubernetesClientMock) GetServiceDetails(sessionCtx context.Context, namespace string) ([]k8s.ServiceDetails, error) {
	originalServices, err := m.InternalKubernetesClient.GetServiceDetails(sessionCtx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get service details: %w", err)
	}

	for i := range originalServices {
		originalServices[i].ClusterIP = "127.0.0.1"
		originalServices[i].HTTPPort = 8080
		originalServices[i].IsHTTPS = false
	}

	return originalServices, nil
}

// GetServiceDetailsByName overrides to simulate local service access
func (m *InternalKubernetesClientMock) GetServiceDetailsByName(sessionCtx context.Context, namespace, serviceName string, serviceType string) (k8s.ServiceDetails, error) {
	originalService, err := m.InternalKubernetesClient.GetServiceDetailsByName(sessionCtx, namespace, serviceName, serviceType)
	if err != nil {
		return k8s.ServiceDetails{}, fmt.Errorf("failed to get service details: %w", err)
	}
	originalService.ClusterIP = "127.0.0.1"
	originalService.HTTPPort = 8080
	originalService.IsHTTPS = false
	return originalService, nil
}

// BearerToken always returns a fake token for tests
func (m *InternalKubernetesClientMock) BearerToken() (string, error) {
	return "FAKE-BEARER-TOKEN", nil
}

func (kc *InternalKubernetesClientMock) GetGroups(ctx context.Context) ([]string, error) {
	return []string{"dora-group-mock", "bella-group-mock"}, nil
}

// CanListServicesInNamespace bypasses SubjectAccessReview for the fake clientset.
// The fake.NewSimpleClientset() doesn't evaluate RBAC rules, so SAR always
// returns "not allowed". In mock mode we grant full access.
func (m *InternalKubernetesClientMock) CanListServicesInNamespace(_ context.Context, _ *k8s.RequestIdentity, _ string) (bool, error) {
	return true, nil
}

// CanAccessServiceInNamespace bypasses SAR for the fake clientset.
func (m *InternalKubernetesClientMock) CanAccessServiceInNamespace(_ context.Context, _ *k8s.RequestIdentity, _, _ string) (bool, error) {
	return true, nil
}

// GetSelfSubjectRulesReview returns all service names for the fake clientset.
func (m *InternalKubernetesClientMock) GetSelfSubjectRulesReview(_ context.Context, _ *k8s.RequestIdentity, _ string) ([]string, error) {
	return []string{}, nil
}
