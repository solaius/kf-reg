package governance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPromotionHandler(t *testing.T) *PromotionActionHandler {
	t.Helper()
	db := newTestDB(t)

	vs := NewVersionStore(db)
	require.NoError(t, vs.AutoMigrate())
	bs := NewBindingStore(db)
	require.NoError(t, bs.AutoMigrate())
	gs := NewGovernanceStore(db)
	as := NewAuditStore(db)

	return NewPromotionActionHandler(gs, vs, bs, as)
}

func TestPromotionActionHandler_VersionCreate(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create a version.
	result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "version.create", result.Action)
	assert.Equal(t, "completed", result.Status)
	assert.NotEmpty(t, result.Data["versionId"])
	assert.Equal(t, "v1.0", result.Data["versionLabel"])

	// Verify version is retrievable.
	versionID := result.Data["versionId"].(string)
	got, err := h.versionStore.GetVersion(versionID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "v1.0", got.VersionLabel)
	assert.Equal(t, "alice", got.CreatedBy)
}

func TestPromotionActionHandler_VersionCreate_DryRun(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, true)
	require.NoError(t, err)
	assert.Equal(t, "dry-run", result.Status)

	// Verify nothing was actually created.
	versions, _, total, err := h.versionStore.ListVersions("mcp:mcpserver:filesystem", 10, "")
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, versions)
}

func TestPromotionActionHandler_VersionCreate_MissingLabel(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	_, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "versionLabel")
}

func TestPromotionActionHandler_BindToDev(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create a version first.
	vResult, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	versionID := vResult.Data["versionId"].(string)

	// Bind to dev (asset is draft, which is allowed for dev).
	result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "dev",
		"versionId":   versionID,
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "promotion.bind", result.Action)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, "dev", result.Data["environment"])
	assert.Equal(t, versionID, result.Data["versionId"])
}

func TestPromotionActionHandler_BindToProdWhileDraft(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create a version first.
	vResult, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	versionID := vResult.Data["versionId"].(string)

	// Try to bind to prod while draft -- should fail.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "prod",
		"versionId":   versionID,
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "draft assets cannot be bound to stage/prod")
}

func TestPromotionActionHandler_BindWhileArchived(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create governance record with archived state.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)

	// Manually update to archived state. We need to go through the normal flow:
	// draft -> approved -> archived. Let's just update the record directly.
	rec, err := h.govStore.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	rec.LifecycleState = string(StateArchived)
	require.NoError(t, h.govStore.Upsert(rec))

	// Create a version.
	vResult, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	versionID := vResult.Data["versionId"].(string)

	// Try to bind to any env while archived -- should fail.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "dev",
		"versionId":   versionID,
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived assets cannot be bound")
}

func TestPromotionActionHandler_PromoteDevToStage(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Set up: create governance record in approved state.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)
	rec, err := h.govStore.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	rec.LifecycleState = string(StateApproved)
	require.NoError(t, h.govStore.Upsert(rec))

	// Create a version.
	vResult, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	versionID := vResult.Data["versionId"].(string)

	// Bind to dev.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "dev",
		"versionId":   versionID,
	}, false)
	require.NoError(t, err)

	// Promote from dev to stage.
	result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.promote", map[string]any{
		"fromEnv": "dev",
		"toEnv":   "stage",
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "promotion.promote", result.Action)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, versionID, result.Data["versionId"])

	// Verify stage binding exists.
	binding, err := h.bindingStore.GetBinding("default", "mcp", "mcpserver", "filesystem", "stage")
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, versionID, binding.VersionID)
}

func TestPromotionActionHandler_RollbackProd(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Set up approved state.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)
	rec, err := h.govStore.Get("default", "mcp", "mcpserver", "filesystem")
	require.NoError(t, err)
	rec.LifecycleState = string(StateApproved)
	require.NoError(t, h.govStore.Upsert(rec))

	// Create two versions.
	v1Result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v1.0",
	}, false)
	require.NoError(t, err)
	v1ID := v1Result.Data["versionId"].(string)

	v2Result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "version.create", map[string]any{
		"versionLabel": "v2.0",
	}, false)
	require.NoError(t, err)
	v2ID := v2Result.Data["versionId"].(string)

	// Bind v1 to prod.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "prod",
		"versionId":   v1ID,
	}, false)
	require.NoError(t, err)

	// Bind v2 to prod (updates binding).
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "prod",
		"versionId":   v2ID,
	}, false)
	require.NoError(t, err)

	// Verify current binding is v2.
	binding, err := h.bindingStore.GetBinding("default", "mcp", "mcpserver", "filesystem", "prod")
	require.NoError(t, err)
	assert.Equal(t, v2ID, binding.VersionID)
	assert.Equal(t, v1ID, binding.PreviousVersionID)

	// Rollback to v1.
	result, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.rollback", map[string]any{
		"environment":     "prod",
		"targetVersionId": v1ID,
	}, false)
	require.NoError(t, err)
	assert.Equal(t, "promotion.rollback", result.Action)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, v1ID, result.Data["versionId"])
	assert.Equal(t, v2ID, result.Data["previousVersionId"])

	// Verify binding was updated.
	binding, err = h.bindingStore.GetBinding("default", "mcp", "mcpserver", "filesystem", "prod")
	require.NoError(t, err)
	assert.Equal(t, v1ID, binding.VersionID)
	assert.Equal(t, v2ID, binding.PreviousVersionID)
}

func TestPromotionActionHandler_UnknownAction(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	_, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "unknown.action", map[string]any{}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown promotion action")
}

func TestPromotionActionHandler_PromoteSameEnv(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	_, err := h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.promote", map[string]any{
		"fromEnv": "dev",
		"toEnv":   "dev",
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fromEnv and toEnv must be different")
}

func TestPromotionActionHandler_BindNonexistentVersion(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create governance record.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)

	// Try binding a version that doesn't exist.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.bind", map[string]any{
		"environment": "dev",
		"versionId":   "nonexistent:version",
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPromotionActionHandler_PromoteNoSourceBinding(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create governance record.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)

	// Try promoting without a source binding.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.promote", map[string]any{
		"fromEnv": "dev",
		"toEnv":   "stage",
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no binding found")
}

func TestPromotionActionHandler_RollbackNonexistentVersion(t *testing.T) {
	h := newTestPromotionHandler(t)
	ctx := context.Background()

	// Create governance record.
	_, err := h.govStore.EnsureExists("default", "mcp", "mcpserver", "filesystem", "mcp:mcpserver:filesystem", "alice")
	require.NoError(t, err)

	// Try rolling back to a nonexistent version.
	_, err = h.HandleAction(ctx, "default", "mcp", "mcpserver", "filesystem", "alice", "promotion.rollback", map[string]any{
		"environment":     "prod",
		"targetVersionId": "nonexistent:version",
	}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
