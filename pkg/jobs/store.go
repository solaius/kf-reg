package jobs

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JobStore provides database operations for refresh jobs.
type JobStore struct {
	db *gorm.DB
}

// NewJobStore creates a new JobStore.
func NewJobStore(db *gorm.DB) *JobStore {
	return &JobStore{db: db}
}

// AutoMigrate creates or updates the refresh_jobs table.
func (s *JobStore) AutoMigrate() error {
	return s.db.AutoMigrate(&RefreshJob{})
}

// JobListFilter defines filters for listing jobs.
type JobListFilter struct {
	Namespace string
	Plugin    string
	SourceID  string
	State     string
	RequestedBy string
}

// Enqueue creates a new queued job. If idempotencyKey is non-empty and a
// non-terminal job with the same key exists, the existing job is returned
// instead of creating a duplicate. Safe for concurrent use.
func (s *JobStore) Enqueue(job *RefreshJob) (*RefreshJob, error) {
	if job.Namespace == "" {
		job.Namespace = "default"
	}
	if job.State == "" {
		job.State = JobStateQueued
	}

	if job.IdempotencyKey == "" {
		if err := s.db.Create(job).Error; err != nil {
			return nil, fmt.Errorf("enqueue job: %w", err)
		}
		return job, nil
	}

	// With idempotency key: use a transaction for atomicity.
	var result *RefreshJob
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Check for existing non-terminal job with this key.
		var existing RefreshJob
		err := tx.Where("idempotency_key = ? AND state IN ?", job.IdempotencyKey,
			[]JobState{JobStateQueued, JobStateRunning}).First(&existing).Error
		if err == nil {
			result = &existing
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("check idempotency key: %w", err)
		}

		// Clear the idempotency key on any terminal jobs with the same key
		// so the unique index doesn't block creating a new job.
		tx.Model(&RefreshJob{}).
			Where("idempotency_key = ? AND state IN ?", job.IdempotencyKey,
				[]JobState{JobStateSucceeded, JobStateFailed, JobStateCanceled}).
			Update("idempotency_key", "")

		if err := tx.Create(job).Error; err != nil {
			// Handle race condition: another transaction may have created the
			// job between our check and create. Look up the existing job.
			var raceExisting RefreshJob
			lookupErr := s.db.Where("idempotency_key = ? AND state IN ?", job.IdempotencyKey,
				[]JobState{JobStateQueued, JobStateRunning}).First(&raceExisting).Error
			if lookupErr == nil {
				result = &raceExisting
				return nil
			}
			return fmt.Errorf("enqueue job: %w", err)
		}
		result = job
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Claim atomically picks a queued job and transitions it to running.
// Uses FOR UPDATE SKIP LOCKED where supported (PostgreSQL).
// Returns nil if no jobs are available.
func (s *JobStore) Claim(maxRetries int) (*RefreshJob, error) {
	var job RefreshJob

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Attempt FOR UPDATE SKIP LOCKED (PostgreSQL).
		// For SQLite or databases that don't support it, fall back to plain SELECT.
		result := tx.Raw(`
			SELECT * FROM refresh_jobs
			WHERE state = ? AND attempt_count <= ?
			ORDER BY requested_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		`, JobStateQueued, maxRetries).Scan(&job)

		if result.Error != nil {
			// Fall back to plain query if FOR UPDATE SKIP LOCKED is not supported.
			result = tx.Where("state = ? AND attempt_count <= ?", JobStateQueued, maxRetries).
				Order("requested_at ASC").
				Limit(1).
				First(&job)
			if result.Error != nil {
				if result.Error == gorm.ErrRecordNotFound {
					return nil
				}
				return result.Error
			}
		}

		if job.ID == "" {
			return nil
		}

		// Transition to running.
		now := time.Now()
		return tx.Model(&RefreshJob{}).Where("id = ? AND state = ?", job.ID, JobStateQueued).
			Updates(map[string]any{
				"state":         JobStateRunning,
				"started_at":    now,
				"attempt_count": gorm.Expr("attempt_count + 1"),
			}).Error
	})

	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	if job.ID == "" {
		return nil, nil
	}

	// Reload to get the updated values.
	if err := s.db.First(&job, "id = ?", job.ID).Error; err != nil {
		return nil, fmt.Errorf("reload claimed job: %w", err)
	}

	return &job, nil
}

// Complete marks a job as succeeded.
func (s *JobStore) Complete(jobID string, entitiesLoaded, entitiesRemoved int, durationMs int64) error {
	now := time.Now()
	result := s.db.Model(&RefreshJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"state":            JobStateSucceeded,
		"finished_at":      now,
		"entities_loaded":  entitiesLoaded,
		"entities_removed": entitiesRemoved,
		"duration_ms":      durationMs,
		"message":          fmt.Sprintf("Loaded %d entities, removed %d", entitiesLoaded, entitiesRemoved),
	})
	if result.Error != nil {
		return fmt.Errorf("complete job: %w", result.Error)
	}
	return nil
}

// Fail marks a job as failed. If the attempt count is within retries, it
// re-queues the job for retry.
func (s *JobStore) Fail(jobID string, errMsg string, maxRetries int) error {
	now := time.Now()

	var job RefreshJob
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		return fmt.Errorf("load job for fail: %w", err)
	}

	updates := map[string]any{
		"last_error":  errMsg,
		"finished_at": now,
	}

	if job.AttemptCount < maxRetries {
		// Re-queue for retry.
		updates["state"] = JobStateQueued
		updates["started_at"] = nil
		updates["finished_at"] = nil
	} else {
		updates["state"] = JobStateFailed
		updates["message"] = "Max retries exceeded: " + errMsg
	}

	result := s.db.Model(&RefreshJob{}).Where("id = ?", jobID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("fail job: %w", result.Error)
	}
	return nil
}

// Cancel marks a queued job as canceled. Running jobs cannot be canceled
// through this method (best-effort cancellation is handled by the worker).
func (s *JobStore) Cancel(jobID string) error {
	now := time.Now()
	result := s.db.Model(&RefreshJob{}).
		Where("id = ? AND state = ?", jobID, JobStateQueued).
		Updates(map[string]any{
			"state":      JobStateCanceled,
			"finished_at": now,
			"message":    "Canceled by user",
		})
	if result.Error != nil {
		return fmt.Errorf("cancel job: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		// Check if the job exists.
		var job RefreshJob
		if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("job not found: %s", jobID)
			}
			return fmt.Errorf("check job: %w", err)
		}
		return fmt.Errorf("job %s is in state %s, only queued jobs can be canceled", jobID, job.State)
	}
	return nil
}

// Get retrieves a job by ID.
func (s *JobStore) Get(jobID string) (*RefreshJob, error) {
	var job RefreshJob
	if err := s.db.First(&job, "id = ?", jobID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

// List returns paginated jobs matching the given filter.
func (s *JobStore) List(filter JobListFilter, pageSize int, pageToken string) ([]RefreshJob, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	buildQuery := func(base *gorm.DB) *gorm.DB {
		q := base.Model(&RefreshJob{})
		if filter.Namespace != "" {
			q = q.Where("namespace = ?", filter.Namespace)
		}
		if filter.Plugin != "" {
			q = q.Where("plugin = ?", filter.Plugin)
		}
		if filter.SourceID != "" {
			q = q.Where("source_id = ?", filter.SourceID)
		}
		if filter.State != "" {
			q = q.Where("state = ?", filter.State)
		}
		if filter.RequestedBy != "" {
			q = q.Where("requested_by = ?", filter.RequestedBy)
		}
		return q
	}

	var totalSize int64
	if err := buildQuery(s.db).Count(&totalSize).Error; err != nil {
		return nil, "", 0, fmt.Errorf("count jobs: %w", err)
	}

	query := buildQuery(s.db).Order("requested_at DESC").Limit(pageSize + 1)
	if pageToken != "" {
		t, err := time.Parse(time.RFC3339Nano, pageToken)
		if err != nil {
			return nil, "", 0, fmt.Errorf("invalid page token: %w", err)
		}
		query = query.Where("requested_at < ?", t)
	}

	var records []RefreshJob
	if err := query.Find(&records).Error; err != nil {
		return nil, "", 0, fmt.Errorf("list jobs: %w", err)
	}

	var nextToken string
	if len(records) > pageSize {
		nextToken = records[pageSize-1].RequestedAt.Format(time.RFC3339Nano)
		records = records[:pageSize]
	}

	return records, nextToken, int(totalSize), nil
}

// CleanupStuckJobs transitions running jobs that have been stuck
// (started_at older than claimTimeout) back to queued for retry.
func (s *JobStore) CleanupStuckJobs(claimTimeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-claimTimeout)
	result := s.db.Model(&RefreshJob{}).
		Where("state = ? AND started_at < ?", JobStateRunning, cutoff).
		Updates(map[string]any{
			"state":      JobStateQueued,
			"started_at": nil,
			"last_error": "Timed out (stuck job recovery)",
		})
	if result.Error != nil {
		return 0, fmt.Errorf("cleanup stuck jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// DeleteOlderThan removes terminal jobs older than the given cutoff.
func (s *JobStore) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := s.db.Where("state IN ? AND finished_at < ?",
		[]JobState{JobStateSucceeded, JobStateFailed, JobStateCanceled}, cutoff).
		Delete(&RefreshJob{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete old jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}
