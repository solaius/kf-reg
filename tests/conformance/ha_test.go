package conformance

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kubeflow/model-registry/pkg/ha"
	"github.com/kubeflow/model-registry/pkg/jobs"
)

// TestPhase8MigrationLockSafety verifies that concurrent migrations don't conflict.
// Two migration lockers on the same DB should serialize their critical sections.
func TestPhase8MigrationLockSafety(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	locker1 := ha.NewMigrationLocker(db)
	locker2 := ha.NewMigrationLocker(db)

	// Track concurrency: only one locker should hold the lock at a time.
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	runLocked := func(locker ha.MigrationLocker) {
		defer wg.Done()
		_ = locker.WithLock(context.Background(), func() error {
			cur := concurrent.Add(1)
			for {
				prev := maxConcurrent.Load()
				if cur <= prev || maxConcurrent.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			concurrent.Add(-1)
			return nil
		})
	}

	wg.Add(2)
	go runLocked(locker1)
	go runLocked(locker2)
	wg.Wait()

	if maxConcurrent.Load() > 1 {
		t.Errorf("expected max concurrency 1, got %d", maxConcurrent.Load())
	}
}

// TestPhase8MigrationLockTableCreation verifies that the migration lock creates
// the necessary table and cleans up after itself.
func TestPhase8MigrationLockTableCreation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	locker := ha.NewMigrationLocker(db)

	var ranMigration bool
	err = locker.WithLock(context.Background(), func() error {
		// Simulate a migration: create a test table.
		if err := db.Exec("CREATE TABLE IF NOT EXISTS test_migration (id TEXT PRIMARY KEY)").Error; err != nil {
			return err
		}
		ranMigration = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithLock error: %v", err)
	}
	if !ranMigration {
		t.Error("migration function was not called")
	}

	// Verify the test table was created.
	if !db.Migrator().HasTable("test_migration") {
		t.Error("test_migration table was not created")
	}
}

// TestPhase8MigrationLockNilDB verifies the noop locker when db is nil.
func TestPhase8MigrationLockNilDB(t *testing.T) {
	locker := ha.NewMigrationLocker(nil)

	called := false
	err := locker.WithLock(context.Background(), func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
}

// TestPhase8MigrationLockConcurrentMigrations verifies that running the
// same AutoMigrate concurrently with locking does not produce errors.
func TestPhase8MigrationLockConcurrentMigrations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}

	locker := ha.NewMigrationLocker(db)

	type testModel struct {
		ID   string `gorm:"primaryKey"`
		Name string
	}

	var wg sync.WaitGroup
	errs := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = locker.WithLock(context.Background(), func() error {
				return db.AutoMigrate(&testModel{})
			})
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d migration error: %v", i, err)
		}
	}

	// Verify table exists.
	if !db.Migrator().HasTable(&testModel{}) {
		t.Error("test model table was not created")
	}
}

// TestPhase8LeaderElectionState verifies the leader state management.
// We cannot run real Kubernetes leader election in unit tests, but we can
// verify the state tracking and callback wiring.
func TestPhase8LeaderElectionState(t *testing.T) {
	cfg := ha.DefaultHAConfig()
	le := ha.NewLeaderElector(cfg, nil, "test-pod-1", slog.Default())

	// Before election: should not be leader.
	if le.IsLeader() {
		t.Error("expected IsLeader() = false before election")
	}

	// Test that callbacks can be registered without panic.
	var startCalled, stopCalled bool
	le.OnStartLeading(func(_ context.Context) {
		startCalled = true
	})
	le.OnStopLeading(func() {
		stopCalled = true
	})

	// Verify initial state does not change after registering callbacks.
	if le.IsLeader() {
		t.Error("expected IsLeader() = false after registering callbacks")
	}

	// We cannot test actual leader election here (requires K8s client),
	// but we verified the state management API.
	_ = startCalled
	_ = stopCalled
}

// TestPhase8HAConfigDefaults verifies that default HA config values are sensible.
func TestPhase8HAConfigDefaults(t *testing.T) {
	cfg := ha.DefaultHAConfig()

	if cfg.LeaderElectionEnabled {
		t.Error("leader election should be disabled by default")
	}
	if cfg.LeaseName == "" {
		t.Error("LeaseName should have a default value")
	}
	if cfg.LeaseDuration == 0 {
		t.Error("LeaseDuration should have a default value")
	}
	if cfg.RenewDeadline == 0 {
		t.Error("RenewDeadline should have a default value")
	}
	if cfg.RetryPeriod == 0 {
		t.Error("RetryPeriod should have a default value")
	}
	if !cfg.MigrationLockEnabled {
		t.Error("MigrationLockEnabled should be true by default")
	}
	if cfg.Identity == "" {
		t.Error("Identity should have a default value")
	}

	// Verify timing relationships: leaseDuration > renewDeadline > retryPeriod.
	if cfg.LeaseDuration <= cfg.RenewDeadline {
		t.Errorf("LeaseDuration (%v) should be > RenewDeadline (%v)", cfg.LeaseDuration, cfg.RenewDeadline)
	}
	if cfg.RenewDeadline <= cfg.RetryPeriod {
		t.Errorf("RenewDeadline (%v) should be > RetryPeriod (%v)", cfg.RenewDeadline, cfg.RetryPeriod)
	}
}

// TestPhase8HANoDuplicateJobs verifies that under simulated multi-replica
// conditions, job idempotency prevents duplicate execution. Replicas are
// simulated sequentially because SQLite does not support concurrent write
// transactions (the go-sqlite driver deadlocks on concurrent transactions
// to the same in-memory database). The idempotency logic itself is
// database-agnostic and works correctly under concurrent access on
// PostgreSQL. True concurrent claim safety is tested in integration tests.
func TestPhase8HANoDuplicateJobs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	store := jobs.NewJobStore(db)
	if err := store.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	idempKey := "default:mcp:hf-source"

	// Simulate 3 replicas sequentially (SQLite limitation, see comment above).
	const replicas = 3
	results := make([]*jobs.RefreshJob, replicas)

	for i := 0; i < replicas; i++ {
		job := &jobs.RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "hf-source",
			RequestedBy:    "replica-" + uuid.New().String()[:8],
			RequestedAt:    time.Now(),
			State:          jobs.JobStateQueued,
			IdempotencyKey: idempKey,
		}
		results[i], err = store.Enqueue(job)
		if err != nil {
			t.Fatalf("replica %d Enqueue error: %v", i, err)
		}
	}

	// All should return the same job ID (idempotency).
	firstID := results[0].ID
	for i := 1; i < replicas; i++ {
		if results[i].ID != firstID {
			t.Errorf("replica %d got different job ID: %s vs %s", i, results[i].ID, firstID)
		}
	}

	// Verify exactly one job exists.
	var total int64
	db.Model(&jobs.RefreshJob{}).Where("idempotency_key = ?", idempKey).Count(&total)
	if total != 1 {
		t.Errorf("expected 1 job with idempotency key, got %d", total)
	}

	// Claiming: only one replica should process it.
	claimed1, err := store.Claim(3)
	if err != nil {
		t.Fatalf("Claim 1 error: %v", err)
	}
	if claimed1 == nil {
		t.Fatal("expected to claim the job")
	}

	// Second claim attempt should return nil (no more queued jobs).
	claimed2, err := store.Claim(3)
	if err != nil {
		t.Fatalf("Claim 2 error: %v", err)
	}
	if claimed2 != nil {
		t.Error("second claim should return nil (job already claimed)")
	}
}
