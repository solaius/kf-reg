package k8mocks

import (
	"context"
	"log/slog"

	"k8s.io/client-go/kubernetes"
)

// SetupFakeClientset populates a fake.NewSimpleClientset with the same mock
// data that SetupEnvTest would create.  This allows the BFF to run in mock
// mode on platforms where envtest binaries (etcd, kube-apiserver) are not
// available (e.g. Windows).
func SetupFakeClientset(clientset kubernetes.Interface, logger *slog.Logger) error {
	ctx := context.Background()
	logger.Info("Populating fake clientset with mock data")
	return setupMock(clientset, ctx)
}
