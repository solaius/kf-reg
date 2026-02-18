package governance

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticProvenanceExtractor(t *testing.T) {
	extractor := &StaticProvenanceExtractor{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "my-source",
		RevisionID: "abc123def456",
	}

	prov := extractor.ExtractProvenance("mcp", "mcpserver", "filesystem")
	require.NotNil(t, prov)
	assert.Equal(t, "git", prov.SourceType)
	assert.Equal(t, "https://github.com/org/repo.git", prov.SourceURI)
	assert.Equal(t, "my-source", prov.SourceID)
	assert.Equal(t, "abc123def456", prov.RevisionID)
}

func TestStaticProvenanceExtractor_IgnoresAssetIdentity(t *testing.T) {
	extractor := &StaticProvenanceExtractor{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "source-1",
		RevisionID: "sha256:deadbeef",
	}

	// Different assets should all get the same provenance.
	prov1 := extractor.ExtractProvenance("mcp", "mcpserver", "server-a")
	prov2 := extractor.ExtractProvenance("knowledge", "dataset", "my-dataset")

	require.NotNil(t, prov1)
	require.NotNil(t, prov2)
	assert.Equal(t, prov1.SourceType, prov2.SourceType)
	assert.Equal(t, prov1.SourceURI, prov2.SourceURI)
	assert.Equal(t, prov1.SourceID, prov2.SourceID)
	assert.Equal(t, prov1.RevisionID, prov2.RevisionID)
}

func TestContentHashProvenanceExtractor(t *testing.T) {
	extractor := &ContentHashProvenanceExtractor{
		SourceType: "yaml",
		SourceURI:  "/etc/catalog/sources.yaml",
		SourceID:   "local-config",
	}

	prov := extractor.ExtractProvenance("mcp", "mcpserver", "filesystem")
	require.NotNil(t, prov)
	assert.Equal(t, "yaml", prov.SourceType)
	assert.Equal(t, "/etc/catalog/sources.yaml", prov.SourceURI)
	assert.Equal(t, "local-config", prov.SourceID)
	assert.Equal(t, "", prov.RevisionID, "ContentHashProvenanceExtractor should not set RevisionID")
}

func TestApplyProvenance_Nil(t *testing.T) {
	record := &AssetVersionRecord{
		ID: "test-id",
	}

	applyProvenance(record, nil)

	assert.Equal(t, "", record.ProvenanceSourceType)
	assert.Equal(t, "", record.ProvenanceSourceURI)
	assert.Equal(t, "", record.ProvenanceSourceID)
	assert.Equal(t, "", record.ProvenanceRevisionID)
}

func TestApplyProvenance_PopulatesFields(t *testing.T) {
	record := &AssetVersionRecord{
		ID: "test-id",
	}

	prov := &ProvenanceInfo{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "my-source",
		RevisionID: "abc123",
	}

	applyProvenance(record, prov)

	assert.Equal(t, "git", record.ProvenanceSourceType)
	assert.Equal(t, "https://github.com/org/repo.git", record.ProvenanceSourceURI)
	assert.Equal(t, "my-source", record.ProvenanceSourceID)
	assert.Equal(t, "abc123", record.ProvenanceRevisionID)
}

func TestApplyProvenance_PartialFields(t *testing.T) {
	record := &AssetVersionRecord{
		ID: "test-id",
	}

	prov := &ProvenanceInfo{
		SourceType: "http",
		SourceURI:  "https://example.com/catalog.yaml",
	}

	applyProvenance(record, prov)

	assert.Equal(t, "http", record.ProvenanceSourceType)
	assert.Equal(t, "https://example.com/catalog.yaml", record.ProvenanceSourceURI)
	assert.Equal(t, "", record.ProvenanceSourceID)
	assert.Equal(t, "", record.ProvenanceRevisionID)
}

func TestProvenanceExtractorInterface(t *testing.T) {
	// Verify all types satisfy the interface at compile time.
	var _ ProvenanceExtractor = &StaticProvenanceExtractor{}
	var _ ProvenanceExtractor = &ContentHashProvenanceExtractor{}
	var _ ProvenanceExtractor = &VerifyingProvenanceExtractor{}
}

func TestApplyProvenance_WithIntegrity(t *testing.T) {
	record := &AssetVersionRecord{
		ID: "test-id",
	}

	prov := &ProvenanceInfo{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "my-source",
		RevisionID: "abc123",
		Integrity: &IntegrityInfo{
			Verified: true,
			Method:   "sha256-content-hash",
			Details:  "sha256:deadbeef1234",
		},
	}

	applyProvenance(record, prov)

	assert.Equal(t, "git", record.ProvenanceSourceType)
	assert.Equal(t, "https://github.com/org/repo.git", record.ProvenanceSourceURI)
	assert.Equal(t, "my-source", record.ProvenanceSourceID)
	assert.Equal(t, "abc123", record.ProvenanceRevisionID)
	assert.True(t, record.ProvenanceVerified)
	assert.Equal(t, "sha256-content-hash", record.ProvenanceMethod)
	assert.Equal(t, "sha256:deadbeef1234", record.ProvenanceDetails)
}

func TestApplyProvenance_WithoutIntegrity(t *testing.T) {
	record := &AssetVersionRecord{
		ID: "test-id",
	}

	prov := &ProvenanceInfo{
		SourceType: "yaml",
		SourceURI:  "/etc/config.yaml",
	}

	applyProvenance(record, prov)

	assert.False(t, record.ProvenanceVerified)
	assert.Equal(t, "", record.ProvenanceMethod)
	assert.Equal(t, "", record.ProvenanceDetails)
}

func TestVerifyingProvenanceExtractor_Success(t *testing.T) {
	inner := &StaticProvenanceExtractor{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "source-1",
		RevisionID: "abc123",
	}

	extractor := &VerifyingProvenanceExtractor{
		Inner:  inner,
		Method: "sha256-content-hash",
		ComputeHash: func(plugin, kind, name string) (string, error) {
			return "sha256:deadbeef", nil
		},
	}

	prov := extractor.ExtractProvenance("mcp", "mcpserver", "test-server")
	require.NotNil(t, prov)
	assert.Equal(t, "git", prov.SourceType)
	assert.Equal(t, "https://github.com/org/repo.git", prov.SourceURI)
	assert.Equal(t, "abc123", prov.RevisionID)

	require.NotNil(t, prov.Integrity)
	assert.True(t, prov.Integrity.Verified)
	assert.Equal(t, "sha256-content-hash", prov.Integrity.Method)
	assert.Equal(t, "sha256:deadbeef", prov.Integrity.Details)
}

func TestVerifyingProvenanceExtractor_HashError(t *testing.T) {
	inner := &StaticProvenanceExtractor{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "source-1",
		RevisionID: "abc123",
	}

	extractor := &VerifyingProvenanceExtractor{
		Inner:  inner,
		Method: "sha256-content-hash",
		ComputeHash: func(plugin, kind, name string) (string, error) {
			return "", errors.New("content not available")
		},
	}

	prov := extractor.ExtractProvenance("mcp", "mcpserver", "test-server")
	require.NotNil(t, prov)

	require.NotNil(t, prov.Integrity)
	assert.False(t, prov.Integrity.Verified)
	assert.Equal(t, "sha256-content-hash", prov.Integrity.Method)
	assert.Equal(t, "content not available", prov.Integrity.Details)
}

func TestVerifyingProvenanceExtractor_NilInner(t *testing.T) {
	inner := &ContentHashProvenanceExtractor{
		SourceType: "yaml",
		SourceURI:  "/config.yaml",
		SourceID:   "local",
	}

	// Test that wrapping works even when inner returns non-nil.
	extractor := &VerifyingProvenanceExtractor{
		Inner:  inner,
		Method: "sha256-content-hash",
		ComputeHash: func(plugin, kind, name string) (string, error) {
			return "sha256:abcd1234", nil
		},
	}

	prov := extractor.ExtractProvenance("mcp", "mcpserver", "test")
	require.NotNil(t, prov)
	assert.Equal(t, "yaml", prov.SourceType)
	require.NotNil(t, prov.Integrity)
	assert.True(t, prov.Integrity.Verified)
}

func TestPromotionActionHandler_VersionCreateWithVerifiedProvenance(t *testing.T) {
	h := newTestPromotionHandler(t)
	h.SetProvenanceExtractor(&VerifyingProvenanceExtractor{
		Inner: &StaticProvenanceExtractor{
			SourceType: "git",
			SourceURI:  "https://github.com/org/repo.git",
			SourceID:   "verified-source",
			RevisionID: "abc123def456",
		},
		Method: "sha256-content-hash",
		ComputeHash: func(plugin, kind, name string) (string, error) {
			return "sha256:verified-hash", nil
		},
	})

	ctx := context.Background()
	result, err := h.HandleAction(ctx, "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v2.0-verified",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)

	versionID := result.Data["versionId"].(string)
	got, err := h.versionStore.GetVersion(versionID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "git", got.ProvenanceSourceType)
	assert.Equal(t, "https://github.com/org/repo.git", got.ProvenanceSourceURI)
	assert.Equal(t, "verified-source", got.ProvenanceSourceID)
	assert.Equal(t, "abc123def456", got.ProvenanceRevisionID)
	assert.True(t, got.ProvenanceVerified)
	assert.Equal(t, "sha256-content-hash", got.ProvenanceMethod)
	assert.Equal(t, "sha256:verified-hash", got.ProvenanceDetails)
}

func TestPromotionActionHandler_VersionCreateWithProvenance(t *testing.T) {
	h := newTestPromotionHandler(t)
	h.SetProvenanceExtractor(&StaticProvenanceExtractor{
		SourceType: "git",
		SourceURI:  "https://github.com/org/repo.git",
		SourceID:   "git-source-1",
		RevisionID: "abc123def456",
	})

	ctx := context.Background()
	result, err := h.HandleAction(ctx, "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)

	// Retrieve the version and verify provenance fields were populated.
	versionID := result.Data["versionId"].(string)
	got, err := h.versionStore.GetVersion(versionID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "git", got.ProvenanceSourceType)
	assert.Equal(t, "https://github.com/org/repo.git", got.ProvenanceSourceURI)
	assert.Equal(t, "git-source-1", got.ProvenanceSourceID)
	assert.Equal(t, "abc123def456", got.ProvenanceRevisionID)
}

func TestPromotionActionHandler_VersionCreateWithoutProvenance(t *testing.T) {
	h := newTestPromotionHandler(t)
	// No provenance extractor set -- provenance fields should remain empty.

	ctx := context.Background()
	result, err := h.HandleAction(ctx, "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)

	versionID := result.Data["versionId"].(string)
	got, err := h.versionStore.GetVersion(versionID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "", got.ProvenanceSourceType)
	assert.Equal(t, "", got.ProvenanceSourceURI)
	assert.Equal(t, "", got.ProvenanceSourceID)
	assert.Equal(t, "", got.ProvenanceRevisionID)
}
