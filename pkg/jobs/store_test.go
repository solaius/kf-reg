package jobs

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RefreshJob{}))
	return db
}

func newTestJob(plugin, sourceID, ns string) *RefreshJob {
	return &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      ns,
		Plugin:         plugin,
		SourceID:       sourceID,
		RequestedBy:    "test-user",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: ns + ":" + plugin + ":" + sourceID,
	}
}

func TestEnqueueCreatesJob(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	created, err := store.Enqueue(job)
	require.NoError(t, err)
	assert.Equal(t, job.ID, created.ID)
	assert.Equal(t, JobStateQueued, created.State)
	assert.Equal(t, "default", created.Namespace)
}

func TestEnqueueDefaultsNamespace(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "")
	created, err := store.Enqueue(job)
	require.NoError(t, err)
	assert.Equal(t, "default", created.Namespace)
}

func TestEnqueueIdempotencyReturnsDuplicate(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job1 := newTestJob("mcp", "src1", "default")
	created1, err := store.Enqueue(job1)
	require.NoError(t, err)

	// Same idempotency key, different ID.
	job2 := newTestJob("mcp", "src1", "default")
	job2.IdempotencyKey = job1.IdempotencyKey
	created2, err := store.Enqueue(job2)
	require.NoError(t, err)

	// Should return the original, not create a new one.
	assert.Equal(t, created1.ID, created2.ID)
}

func TestEnqueueIdempotencyAllowsAfterTerminal(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job1 := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job1)
	require.NoError(t, err)

	// Mark the first job as succeeded.
	require.NoError(t, store.Complete(job1.ID, 5, 0, 100))

	// Now a new job with same idempotency key should be created.
	job2 := newTestJob("mcp", "src1", "default")
	job2.IdempotencyKey = job1.IdempotencyKey
	created2, err := store.Enqueue(job2)
	require.NoError(t, err)
	assert.NotEqual(t, job1.ID, created2.ID)
}

func TestClaimReturnsQueuedJob(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	claimed, err := store.Claim(3)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	assert.Equal(t, job.ID, claimed.ID)
	assert.Equal(t, JobStateRunning, claimed.State)
	assert.NotNil(t, claimed.StartedAt)
	assert.Equal(t, 1, claimed.AttemptCount)
}

func TestClaimReturnsNilWhenEmpty(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	claimed, err := store.Claim(3)
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestClaimRespectsMaxRetries(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	job.AttemptCount = 4 // exceeded max retries of 3
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	claimed, err := store.Claim(3)
	require.NoError(t, err)
	assert.Nil(t, claimed)
}

func TestCompleteUpdatesJob(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	err = store.Complete(job.ID, 10, 2, 5000)
	require.NoError(t, err)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStateSucceeded, result.State)
	assert.Equal(t, 10, result.EntitiesLoaded)
	assert.Equal(t, 2, result.EntitiesRemoved)
	assert.Equal(t, int64(5000), result.DurationMs)
	assert.NotNil(t, result.FinishedAt)
}

func TestFailRequeuesWhenRetriesLeft(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	// Claim the job (sets attempt_count=1).
	_, err = store.Claim(3)
	require.NoError(t, err)

	err = store.Fail(job.ID, "transient error", 3)
	require.NoError(t, err)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStateQueued, result.State, "should re-queue for retry")
	assert.Equal(t, "transient error", result.LastError)
}

func TestFailMarksFailedAtMaxRetries(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	job.AttemptCount = 3 // already at max
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	err = store.Fail(job.ID, "fatal error", 3)
	require.NoError(t, err)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStateFailed, result.State)
	assert.Contains(t, result.Message, "Max retries exceeded")
}

func TestCancelQueuedJobSucceeds(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	err = store.Cancel(job.ID)
	require.NoError(t, err)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStateCanceled, result.State)
}

func TestCancelRunningJobFails(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	// Claim to transition to running.
	_, err = store.Claim(3)
	require.NoError(t, err)

	err = store.Cancel(job.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "running")
}

func TestCancelNonExistentJobFails(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	err := store.Cancel("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetReturnsNilForMissing(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job, err := store.Get("nonexistent")
	require.NoError(t, err)
	assert.Nil(t, job)
}

func TestListWithFilters(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	// Create jobs for different plugins and namespaces.
	for i, plugin := range []string{"mcp", "mcp", "knowledge"} {
		j := &RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "ns1",
			Plugin:         plugin,
			SourceID:       "src",
			RequestedBy:    "user",
			RequestedAt:    time.Now().Add(time.Duration(i) * time.Second),
			State:          JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		_, err := store.Enqueue(j)
		require.NoError(t, err)
	}

	// Filter by plugin.
	results, _, total, err := store.List(JobListFilter{Plugin: "mcp"}, 10, "")
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, results, 2)

	// Filter by namespace.
	results, _, total, err = store.List(JobListFilter{Namespace: "ns1"}, 10, "")
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, results, 3)
}

func TestListPagination(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	// Create 5 jobs with staggered times.
	for i := 0; i < 5; i++ {
		j := &RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "src",
			RequestedBy:    "user",
			RequestedAt:    time.Now().Add(time.Duration(i) * time.Minute),
			State:          JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		_, err := store.Enqueue(j)
		require.NoError(t, err)
	}

	// First page of 2.
	results, nextToken, total, err := store.List(JobListFilter{}, 2, "")
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 5, total)
	assert.NotEmpty(t, nextToken)

	// Second page.
	results2, nextToken2, _, err := store.List(JobListFilter{}, 2, nextToken)
	require.NoError(t, err)
	assert.Len(t, results2, 2)
	assert.NotEmpty(t, nextToken2)

	// Last page.
	results3, nextToken3, _, err := store.List(JobListFilter{}, 2, nextToken2)
	require.NoError(t, err)
	assert.Len(t, results3, 1)
	assert.Empty(t, nextToken3)
}

func TestCleanupStuckJobs(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	// Claim the job.
	_, err = store.Claim(3)
	require.NoError(t, err)

	// Manually set started_at far in the past.
	oldTime := time.Now().Add(-20 * time.Minute)
	db.Model(&RefreshJob{}).Where("id = ?", job.ID).Update("started_at", oldTime)

	recovered, err := store.CleanupStuckJobs(10 * time.Minute)
	require.NoError(t, err)
	assert.Equal(t, int64(1), recovered)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Equal(t, JobStateQueued, result.State)
}

func TestDeleteOlderThan(t *testing.T) {
	db := setupTestDB(t)
	store := NewJobStore(db)

	job := newTestJob("mcp", "src1", "default")
	_, err := store.Enqueue(job)
	require.NoError(t, err)
	require.NoError(t, store.Complete(job.ID, 1, 0, 100))

	// Set finished_at far in the past.
	oldTime := time.Now().Add(-10 * 24 * time.Hour)
	db.Model(&RefreshJob{}).Where("id = ?", job.ID).Update("finished_at", oldTime)

	deleted, err := store.DeleteOlderThan(time.Now().Add(-7 * 24 * time.Hour))
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	assert.Nil(t, result)
}
