package jobs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// mockRefresher implements PluginRefresher for tests.
type mockRefresher struct {
	refreshErr  error
	loaded      int
	removed     int
	refreshDur  time.Duration
	refreshCalls int
}

func (m *mockRefresher) Refresh(ctx context.Context, sourceID string) (int, int, time.Duration, error) {
	m.refreshCalls++
	if m.refreshErr != nil {
		return 0, 0, 0, m.refreshErr
	}
	return m.loaded, m.removed, m.refreshDur, nil
}

func (m *mockRefresher) RefreshAll(ctx context.Context) (int, int, time.Duration, error) {
	m.refreshCalls++
	if m.refreshErr != nil {
		return 0, 0, 0, m.refreshErr
	}
	return m.loaded, m.removed, m.refreshDur, nil
}

func setupWorkerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use a unique file-based DSN per test to avoid interference from cleanup
	// goroutines that may run after the test completes.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.New().String())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RefreshJob{}))
	return db
}

func TestWorkerProcessesJob(t *testing.T) {
	db := setupWorkerTestDB(t)
	store := NewJobStore(db)

	mock := &mockRefresher{loaded: 5, removed: 1, refreshDur: 100 * time.Millisecond}
	lookup := func(name string) (PluginRefresher, bool) {
		if name == "mcp" {
			return mock, true
		}
		return nil, false
	}

	cfg := DefaultJobConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.Concurrency = 1
	cfg.ClaimTimeout = 0
	cfg.RetentionDays = 0

	wp := NewWorkerPool(store, lookup, cfg, nil)

	// Enqueue a job.
	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go wp.Run(ctx)

	// Wait for the job to be processed.
	require.Eventually(t, func() bool {
		j, _ := store.Get(job.ID)
		return j != nil && j.State == JobStateSucceeded
	}, 2*time.Second, 50*time.Millisecond, "job should be completed")

	result, _ := store.Get(job.ID)
	assert.Equal(t, 5, result.EntitiesLoaded)
	assert.Equal(t, 1, result.EntitiesRemoved)
	assert.Equal(t, 1, mock.refreshCalls)

	cancel()
}

func TestWorkerRetriesOnFailure(t *testing.T) {
	db := setupWorkerTestDB(t)
	store := NewJobStore(db)

	callCount := 0
	mock := &mockRefresher{}
	lookup := func(name string) (PluginRefresher, bool) {
		if name == "mcp" {
			return &failThenSucceedRefresher{failCount: 1, callCount: &callCount}, true
		}
		return nil, false
	}

	cfg := DefaultJobConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.Concurrency = 1
	cfg.MaxRetries = 3
	cfg.ClaimTimeout = 0
	cfg.RetentionDays = 0

	_ = mock
	wp := NewWorkerPool(store, lookup, cfg, nil)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go wp.Run(ctx)

	// Wait for the job to be eventually completed after retry.
	require.Eventually(t, func() bool {
		j, _ := store.Get(job.ID)
		return j != nil && j.State == JobStateSucceeded
	}, 5*time.Second, 100*time.Millisecond, "job should eventually succeed after retry")

	assert.Equal(t, 2, callCount, "should have been called twice (fail + succeed)")

	cancel()
}

func TestWorkerFailsAfterMaxRetries(t *testing.T) {
	db := setupWorkerTestDB(t)
	store := NewJobStore(db)

	mock := &mockRefresher{refreshErr: fmt.Errorf("persistent error"), loaded: 0, removed: 0}
	lookup := func(name string) (PluginRefresher, bool) {
		if name == "mcp" {
			return mock, true
		}
		return nil, false
	}

	cfg := DefaultJobConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.Concurrency = 1
	cfg.MaxRetries = 2
	cfg.ClaimTimeout = 0
	cfg.RetentionDays = 0

	wp := NewWorkerPool(store, lookup, cfg, nil)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go wp.Run(ctx)

	// Wait for the job to be marked as failed.
	require.Eventually(t, func() bool {
		j, _ := store.Get(job.ID)
		return j != nil && j.State == JobStateFailed
	}, 5*time.Second, 100*time.Millisecond, "job should be marked failed after max retries")

	cancel()
}

func TestWorkerRefreshAll(t *testing.T) {
	db := setupWorkerTestDB(t)
	store := NewJobStore(db)

	mock := &mockRefresher{loaded: 10, removed: 0, refreshDur: 50 * time.Millisecond}
	lookup := func(name string) (PluginRefresher, bool) {
		if name == "mcp" {
			return mock, true
		}
		return nil, false
	}

	cfg := DefaultJobConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.Concurrency = 1
	cfg.ClaimTimeout = 0
	cfg.RetentionDays = 0

	wp := NewWorkerPool(store, lookup, cfg, nil)

	// SourceID="_all" triggers RefreshAll.
	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "_all",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go wp.Run(ctx)

	require.Eventually(t, func() bool {
		j, _ := store.Get(job.ID)
		return j != nil && j.State == JobStateSucceeded
	}, 2*time.Second, 50*time.Millisecond)

	result, _ := store.Get(job.ID)
	assert.Equal(t, 10, result.EntitiesLoaded)

	cancel()
}

func TestWorkerUnknownPlugin(t *testing.T) {
	db := setupWorkerTestDB(t)
	store := NewJobStore(db)

	lookup := func(name string) (PluginRefresher, bool) {
		return nil, false
	}

	cfg := DefaultJobConfig()
	cfg.PollInterval = 50 * time.Millisecond
	cfg.Concurrency = 1
	cfg.MaxRetries = 1
	// Disable cleanup to avoid accessing DB after context cancellation.
	cfg.ClaimTimeout = 0
	cfg.RetentionDays = 0

	wp := NewWorkerPool(store, lookup, cfg, nil)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "nonexistent",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go wp.Run(ctx)

	require.Eventually(t, func() bool {
		j, _ := store.Get(job.ID)
		return j != nil && j.State == JobStateFailed
	}, 2*time.Second, 50*time.Millisecond)

	cancel()

	result, err := store.Get(job.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Contains(t, result.LastError, "not found")
}

// failThenSucceedRefresher fails the first N calls, then succeeds.
type failThenSucceedRefresher struct {
	failCount int
	callCount *int
}

func (f *failThenSucceedRefresher) Refresh(ctx context.Context, sourceID string) (int, int, time.Duration, error) {
	*f.callCount++
	if *f.callCount <= f.failCount {
		return 0, 0, 0, fmt.Errorf("transient failure #%d", *f.callCount)
	}
	return 3, 0, 50 * time.Millisecond, nil
}

func (f *failThenSucceedRefresher) RefreshAll(ctx context.Context) (int, int, time.Duration, error) {
	return f.Refresh(ctx, "_all")
}
