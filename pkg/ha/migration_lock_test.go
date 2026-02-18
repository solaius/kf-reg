package ha

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Use shared cache so all goroutines see the same in-memory database.
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func TestNewMigrationLocker_NilDB(t *testing.T) {
	locker := NewMigrationLocker(nil)
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

func TestFallbackMigrationLock_WithLock(t *testing.T) {
	db := setupTestDB(t)
	locker := NewMigrationLocker(db)

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

	// Verify lock was released: lock table should be empty.
	var count int64
	db.Model(&migrationLockRecord{}).Count(&count)
	if count != 0 {
		t.Errorf("expected lock table to be empty after WithLock, got %d rows", count)
	}
}

func TestFallbackMigrationLock_ErrorPropagation(t *testing.T) {
	db := setupTestDB(t)
	locker := NewMigrationLocker(db)

	expectedErr := "migration failed"
	err := locker.WithLock(context.Background(), func() error {
		return &testError{msg: expectedErr}
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expectedErr {
		t.Errorf("error = %q, want %q", err.Error(), expectedErr)
	}

	// Lock should still be released after error.
	var count int64
	db.Model(&migrationLockRecord{}).Count(&count)
	if count != 0 {
		t.Errorf("expected lock table to be empty after error, got %d rows", count)
	}
}

func TestFallbackMigrationLock_Serialization(t *testing.T) {
	db := setupTestDB(t)
	locker := NewMigrationLocker(db)

	// Verify that two concurrent WithLock calls serialize: only one
	// runs the critical section at a time.
	var concurrent atomic.Int32
	var maxConcurrent atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = locker.WithLock(context.Background(), func() error {
				cur := concurrent.Add(1)
				// Track the maximum concurrency observed.
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
		}()
	}

	wg.Wait()

	if maxConcurrent.Load() > 1 {
		t.Errorf("expected max concurrency of 1, got %d", maxConcurrent.Load())
	}
}

func TestFallbackMigrationLock_ContextCancellation(t *testing.T) {
	db := setupTestDB(t)
	locker := NewMigrationLocker(db)

	// Acquire the lock first.
	err := locker.WithLock(context.Background(), func() error {
		// While holding the lock, try to acquire it again with a cancelled context.
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately.

		err2 := locker.WithLock(ctx, func() error {
			t.Error("should not have acquired the lock")
			return nil
		})
		if err2 == nil {
			t.Error("expected context cancellation error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("outer WithLock error: %v", err)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
