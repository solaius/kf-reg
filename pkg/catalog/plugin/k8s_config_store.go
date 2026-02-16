package plugin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"gopkg.in/yaml.v3"
)

const (
	// revisionsAnnotationKey is the ConfigMap annotation key for revision metadata.
	revisionsAnnotationKey = "catalog.kubeflow.org/revisions"

	// revisionDataPrefix is the prefix for ConfigMap annotations storing revision data snapshots.
	revisionDataPrefix = "catalog.kubeflow.org/rev-"

	// maxK8sRevisionHistory is the maximum number of revision snapshots to keep in annotations.
	maxK8sRevisionHistory = 10

	// maxConfigMapDataSize is the approximate size limit for ConfigMap data (900 KiB, leaving
	// headroom within the 1 MiB etcd value limit for metadata and annotations).
	maxConfigMapDataSize = 900 << 10
)

// K8sSourceConfigStore implements ConfigStore backed by a Kubernetes ConfigMap.
// The configuration YAML is stored in a single data key within the ConfigMap.
// Revision metadata and snapshots are stored in ConfigMap annotations.
type K8sSourceConfigStore struct {
	client        kubernetes.Interface
	namespace     string
	configMapName string
	dataKey       string
	mu            sync.Mutex
}

// NewK8sSourceConfigStore creates a new ConfigStore backed by a Kubernetes ConfigMap.
//   - client: a Kubernetes clientset (or fake for testing)
//   - namespace: the namespace containing the ConfigMap
//   - configMapName: the ConfigMap name (e.g., "catalog-sources")
//   - dataKey: the data key within the ConfigMap (e.g., "sources.yaml")
func NewK8sSourceConfigStore(client kubernetes.Interface, namespace, configMapName, dataKey string) *K8sSourceConfigStore {
	return &K8sSourceConfigStore{
		client:        client,
		namespace:     namespace,
		configMapName: configMapName,
		dataKey:       dataKey,
	}
}

// Load reads the current configuration from the ConfigMap and returns it along
// with a version string (SHA-256 hex digest of the data value).
func (s *K8sSourceConfigStore) Load(ctx context.Context) (*CatalogSourcesConfig, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, s.configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, "", fmt.Errorf("k8s config store: failed to get ConfigMap %s/%s: %w",
			s.namespace, s.configMapName, err)
	}

	data, ok := cm.Data[s.dataKey]
	if !ok {
		return nil, "", fmt.Errorf("k8s config store: key %q not found in ConfigMap %s/%s",
			s.dataKey, s.namespace, s.configMapName)
	}

	version := hashContent([]byte(data))

	origin := fmt.Sprintf("configmap:%s/%s[%s]", s.namespace, s.configMapName, s.dataKey)
	cfg, err := ParseConfig([]byte(data), origin)
	if err != nil {
		return nil, "", fmt.Errorf("k8s config store: failed to parse config from ConfigMap: %w", err)
	}

	return cfg, version, nil
}

// Save marshals the configuration to YAML and updates the ConfigMap.
// The provided version must match the current content hash; otherwise
// ErrVersionConflict is returned. On success the new version hash is returned.
func (s *K8sSourceConfigStore) Save(ctx context.Context, cfg *CatalogSourcesConfig, version string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current ConfigMap.
	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, s.configMapName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("k8s config store: failed to get ConfigMap for save: %w", err)
	}

	// Verify optimistic concurrency via content hash.
	currentData := cm.Data[s.dataKey]
	currentVersion := hashContent([]byte(currentData))
	if currentVersion != version {
		return "", ErrVersionConflict
	}

	// Marshal new config.
	newData, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("k8s config store: failed to marshal config: %w", err)
	}

	if len(newData) > maxConfigMapDataSize {
		return "", fmt.Errorf("k8s config store: marshaled config exceeds maximum allowed size")
	}

	// Snapshot current data in annotations before overwriting.
	s.snapshotRevision(cm, currentData, currentVersion)

	// Update the data key.
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[s.dataKey] = string(newData)

	// Update the ConfigMap in Kubernetes.
	_, err = s.client.CoreV1().ConfigMaps(s.namespace).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		if k8serrors.IsConflict(err) {
			return "", ErrVersionConflict
		}
		return "", fmt.Errorf("k8s config store: failed to update ConfigMap: %w", err)
	}

	newVersion := hashContent(newData)
	return newVersion, nil
}

// Watch is not implemented for K8sSourceConfigStore. Returns nil channel and nil error.
// A future implementation could use informers to watch for ConfigMap changes.
func (s *K8sSourceConfigStore) Watch(_ context.Context) (<-chan ConfigChangeEvent, error) {
	return nil, nil
}

// ListRevisions returns the revision history stored in ConfigMap annotations,
// sorted newest first.
func (s *K8sSourceConfigStore) ListRevisions(ctx context.Context) ([]ConfigRevision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, s.configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("k8s config store: failed to get ConfigMap for revisions: %w", err)
	}

	return s.parseRevisions(cm), nil
}

// Rollback restores the configuration to a previous revision identified by
// its version hash. The revision data is read from ConfigMap annotations,
// validated, and then saved via the normal Save path for concurrency safety.
func (s *K8sSourceConfigStore) Rollback(ctx context.Context, version string) (*CatalogSourcesConfig, string, error) {
	// Read revision data under lock, then release for Save (which re-acquires).
	s.mu.Lock()

	cm, err := s.client.CoreV1().ConfigMaps(s.namespace).Get(ctx, s.configMapName, metav1.GetOptions{})
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("k8s config store: failed to get ConfigMap for rollback: %w", err)
	}

	// Find the revision data annotation.
	annotationKey := revisionDataPrefix + version
	revData, ok := cm.Annotations[annotationKey]
	if !ok {
		// Try short version match.
		revData, ok = s.findRevisionDataByPrefix(cm, version)
		if !ok {
			s.mu.Unlock()
			return nil, "", ErrRevisionNotFound
		}
	}

	// Parse the revision data.
	origin := fmt.Sprintf("configmap:%s/%s[%s]", s.namespace, s.configMapName, s.dataKey)
	cfg, err := ParseConfig([]byte(revData), origin)
	if err != nil {
		s.mu.Unlock()
		return nil, "", fmt.Errorf("k8s config store: revision data is invalid: %w", err)
	}

	// Get current version for concurrency check.
	currentData := cm.Data[s.dataKey]
	currentVersion := hashContent([]byte(currentData))
	s.mu.Unlock()

	// Save through the normal path.
	newVersion, err := s.Save(ctx, cfg, currentVersion)
	if err != nil {
		return nil, "", fmt.Errorf("k8s config store: failed to save rolled-back config: %w", err)
	}

	return cfg, newVersion, nil
}

// snapshotRevision stores the current data as a revision in ConfigMap annotations.
// Must be called with s.mu held.
func (s *K8sSourceConfigStore) snapshotRevision(cm *corev1.ConfigMap, data, version string) {
	if cm.Annotations == nil {
		cm.Annotations = make(map[string]string)
	}

	// Use short version (first 8 chars) for annotation keys to save space.
	versionShort := version
	if len(versionShort) > 8 {
		versionShort = versionShort[:8]
	}

	// Store the data snapshot.
	cm.Annotations[revisionDataPrefix+versionShort] = data

	// Add to revision metadata list.
	revisions := s.parseRevisions(cm)
	revisions = append(revisions, ConfigRevision{
		Version:   versionShort,
		Timestamp: time.Now(),
		Size:      int64(len(data)),
	})

	// Sort newest first.
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Timestamp.After(revisions[j].Timestamp)
	})

	// Prune old revisions.
	if len(revisions) > maxK8sRevisionHistory {
		for _, old := range revisions[maxK8sRevisionHistory:] {
			delete(cm.Annotations, revisionDataPrefix+old.Version)
		}
		revisions = revisions[:maxK8sRevisionHistory]
	}

	// Serialize revision metadata.
	revJSON, err := json.Marshal(revisions)
	if err == nil {
		cm.Annotations[revisionsAnnotationKey] = string(revJSON)
	}
}

// parseRevisions reads the revision metadata from ConfigMap annotations.
func (s *K8sSourceConfigStore) parseRevisions(cm *corev1.ConfigMap) []ConfigRevision {
	if cm.Annotations == nil {
		return nil
	}

	raw, ok := cm.Annotations[revisionsAnnotationKey]
	if !ok {
		return nil
	}

	var revisions []ConfigRevision
	if err := json.Unmarshal([]byte(raw), &revisions); err != nil {
		return nil
	}

	return revisions
}

// findRevisionDataByPrefix looks for a revision data annotation matching the
// given version prefix (for short hash lookups).
func (s *K8sSourceConfigStore) findRevisionDataByPrefix(cm *corev1.ConfigMap, prefix string) (string, bool) {
	if cm.Annotations == nil {
		return "", false
	}

	short := prefix
	if len(short) > 8 {
		short = short[:8]
	}

	data, ok := cm.Annotations[revisionDataPrefix+short]
	return data, ok
}

// hashContent returns the SHA-256 hex digest of the given byte slice.
func hashContent(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

// Compile-time interface check.
var _ ConfigStore = (*K8sSourceConfigStore)(nil)
