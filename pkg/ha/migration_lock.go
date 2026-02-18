package ha

import (
	"context"
	"fmt"
	"hash/crc32"
	"os"
	"time"

	"gorm.io/gorm"
)

// MigrationLocker is the interface for acquiring a lock around database
// migrations to prevent concurrent AutoMigrate calls from multiple replicas.
type MigrationLocker interface {
	// WithLock executes fn while holding the migration lock.
	// It blocks until the lock is acquired, then releases it after fn returns.
	WithLock(ctx context.Context, fn func() error) error
}

// NewMigrationLocker creates a MigrationLocker appropriate for the database
// dialect. PostgreSQL uses advisory locks; other databases use a table-based
// fallback. The lock table is created immediately for the fallback strategy.
func NewMigrationLocker(db *gorm.DB) MigrationLocker {
	if db == nil {
		return &noopMigrationLock{}
	}
	dialector := db.Dialector.Name()
	if dialector == "postgres" {
		return &pgAdvisoryLock{
			db:     db,
			lockID: int64(crc32.ChecksumIEEE([]byte("catalog-server-migration"))),
		}
	}
	lock := &fallbackMigrationLock{db: db}
	// Create the lock table immediately so that concurrent callers never
	// hit "no such table" errors on their first WithLock call.
	_ = db.AutoMigrate(&migrationLockRecord{})
	return lock
}

// noopMigrationLock is used when no database is configured.
type noopMigrationLock struct{}

func (n *noopMigrationLock) WithLock(_ context.Context, fn func() error) error {
	return fn()
}

// pgAdvisoryLock uses PostgreSQL advisory locks for migration serialization.
type pgAdvisoryLock struct {
	db     *gorm.DB
	lockID int64
}

func (l *pgAdvisoryLock) WithLock(ctx context.Context, fn func() error) error {
	// Acquire advisory lock (blocks until available).
	if err := l.db.WithContext(ctx).Exec("SELECT pg_advisory_lock(?)", l.lockID).Error; err != nil {
		return fmt.Errorf("failed to acquire migration advisory lock: %w", err)
	}

	// Always release the lock.
	defer func() {
		_ = l.db.Exec("SELECT pg_advisory_unlock(?)", l.lockID).Error
	}()

	return fn()
}

// migrationLockRecord is the table-based lock row for non-PostgreSQL databases.
type migrationLockRecord struct {
	ID       string    `gorm:"primaryKey;column:id"`
	LockedAt time.Time `gorm:"column:locked_at"`
	LockedBy string    `gorm:"column:locked_by"`
}

func (migrationLockRecord) TableName() string { return "migration_lock" }

// fallbackMigrationLock uses a database table for locking on non-PostgreSQL
// databases (SQLite, MySQL). It uses INSERT-or-fail semantics to ensure only
// one holder at a time, with stale lock cleanup for crash recovery.
type fallbackMigrationLock struct {
	db *gorm.DB
}

func (l *fallbackMigrationLock) WithLock(ctx context.Context, fn func() error) error {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	lockRow := migrationLockRecord{
		ID:       "migration",
		LockedBy: hostname,
	}

	const maxRetries = 30
	const retryInterval = 1 * time.Second
	const staleLockAge = 5 * time.Minute

	acquired := false
	for i := 0; i < maxRetries; i++ {
		// Delete stale locks (older than staleLockAge) to handle crash recovery.
		l.db.WithContext(ctx).Where("id = ? AND locked_at < ?", "migration", time.Now().Add(-staleLockAge)).Delete(&migrationLockRecord{})

		// Update lockRow timestamp for each attempt.
		lockRow.LockedAt = time.Now()

		// Try to insert (fails if row already exists).
		result := l.db.WithContext(ctx).Create(&lockRow)
		if result.Error == nil {
			acquired = true
			break
		}

		if i == maxRetries-1 {
			return fmt.Errorf("failed to acquire migration lock after %d retries: %w", maxRetries, result.Error)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}

	if !acquired {
		return fmt.Errorf("failed to acquire migration lock")
	}

	// Always release the lock.
	defer func() {
		l.db.Where("id = ?", "migration").Delete(&migrationLockRecord{})
	}()

	return fn()
}
