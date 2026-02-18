package jobs

import (
	"time"
)

// JobState represents the lifecycle state of a refresh job.
type JobState string

const (
	JobStateQueued    JobState = "queued"
	JobStateRunning   JobState = "running"
	JobStateSucceeded JobState = "succeeded"
	JobStateFailed    JobState = "failed"
	JobStateCanceled  JobState = "canceled"
)

// RefreshJob is the GORM model for a refresh job.
type RefreshJob struct {
	ID              string     `gorm:"primaryKey;column:id;type:varchar(36)"`
	Namespace       string     `gorm:"column:namespace;index:idx_job_ns_state,priority:1;default:default;not null"`
	Plugin          string     `gorm:"column:plugin;index:idx_job_plugin_state,priority:1;not null"`
	SourceID        string     `gorm:"column:source_id"`
	RequestedBy     string     `gorm:"column:requested_by;not null"`
	RequestedAt     time.Time  `gorm:"column:requested_at;not null"`
	State           JobState   `gorm:"column:state;index:idx_job_ns_state,priority:2;index:idx_job_plugin_state,priority:2;index:idx_job_state;not null;default:queued"`
	Progress        string     `gorm:"column:progress"`
	Message         string     `gorm:"column:message"`
	StartedAt       *time.Time `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	AttemptCount    int        `gorm:"column:attempt_count;default:0"`
	LastError       string     `gorm:"column:last_error"`
	IdempotencyKey  string     `gorm:"column:idempotency_key;uniqueIndex:idx_job_idemp_key"`
	EntitiesLoaded  int        `gorm:"column:entities_loaded"`
	EntitiesRemoved int        `gorm:"column:entities_removed"`
	DurationMs      int64      `gorm:"column:duration_ms"`
}

// TableName returns the GORM table name.
func (RefreshJob) TableName() string { return "refresh_jobs" }

// IsTerminal returns true if the job is in a terminal state.
func (j *RefreshJob) IsTerminal() bool {
	switch j.State {
	case JobStateSucceeded, JobStateFailed, JobStateCanceled:
		return true
	}
	return false
}
