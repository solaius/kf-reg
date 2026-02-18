package conformance

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/kubeflow/model-registry/pkg/jobs"
)

// setupJobTestDB creates an in-memory SQLite DB with the refresh_jobs table.
func setupJobTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	store := jobs.NewJobStore(db)
	if err := store.AutoMigrate(); err != nil {
		t.Fatalf("failed to migrate jobs table: %v", err)
	}
	return db
}

// TestPhase8JobLifecycle verifies the full job lifecycle:
// queued -> running -> succeeded.
func TestPhase8JobLifecycle(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	job := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "huggingface",
		RequestedBy:    "alice",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}

	// Enqueue.
	created, err := store.Enqueue(job)
	if err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}
	if created.State != jobs.JobStateQueued {
		t.Errorf("state = %q, want %q", created.State, jobs.JobStateQueued)
	}

	// Claim (queued -> running).
	claimed, err := store.Claim(3)
	if err != nil {
		t.Fatalf("Claim error: %v", err)
	}
	if claimed == nil {
		t.Fatal("Claim returned nil")
	}
	if claimed.State != jobs.JobStateRunning {
		t.Errorf("state after claim = %q, want %q", claimed.State, jobs.JobStateRunning)
	}
	if claimed.AttemptCount != 1 {
		t.Errorf("attemptCount = %d, want 1", claimed.AttemptCount)
	}

	// Complete (running -> succeeded).
	if err := store.Complete(job.ID, 42, 3, 1500); err != nil {
		t.Fatalf("Complete error: %v", err)
	}

	result, err := store.Get(job.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.State != jobs.JobStateSucceeded {
		t.Errorf("state = %q, want %q", result.State, jobs.JobStateSucceeded)
	}
	if result.EntitiesLoaded != 42 {
		t.Errorf("entitiesLoaded = %d, want 42", result.EntitiesLoaded)
	}
	if result.EntitiesRemoved != 3 {
		t.Errorf("entitiesRemoved = %d, want 3", result.EntitiesRemoved)
	}
	if result.DurationMs != 1500 {
		t.Errorf("durationMs = %d, want 1500", result.DurationMs)
	}
}

// TestPhase8JobIdempotency verifies that duplicate idempotency keys return
// the existing job instead of creating a new one.
func TestPhase8JobIdempotency(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	idempKey := "ns:mcp:src1:" + uuid.New().String()

	job1 := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "alice",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: idempKey,
	}
	created1, err := store.Enqueue(job1)
	if err != nil {
		t.Fatalf("first Enqueue error: %v", err)
	}

	// Second enqueue with same idempotency key should return the first.
	job2 := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "bob",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: idempKey,
	}
	created2, err := store.Enqueue(job2)
	if err != nil {
		t.Fatalf("second Enqueue error: %v", err)
	}

	if created1.ID != created2.ID {
		t.Errorf("expected same job ID for duplicate idempotency key, got %s vs %s", created1.ID, created2.ID)
	}

	// After the first job completes, a new job with same key should be allowed.
	if err := store.Complete(created1.ID, 1, 0, 100); err != nil {
		t.Fatalf("Complete error: %v", err)
	}

	job3 := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "charlie",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: idempKey,
	}
	created3, err := store.Enqueue(job3)
	if err != nil {
		t.Fatalf("third Enqueue error: %v", err)
	}

	if created3.ID == created1.ID {
		t.Error("expected new job after terminal state, but got same ID")
	}
}

// TestPhase8JobConcurrency verifies that sequential claims process all queued jobs.
// Note: SQLite does not support FOR UPDATE SKIP LOCKED, so this test validates
// sequential claim behavior. True concurrent claim safety is tested against
// PostgreSQL in integration tests.
func TestPhase8JobConcurrency(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	// Enqueue 10 jobs.
	jobIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		job := &jobs.RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "src",
			RequestedBy:    "test",
			RequestedAt:    time.Now().Add(time.Duration(i) * time.Millisecond),
			State:          jobs.JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		if _, err := store.Enqueue(job); err != nil {
			t.Fatalf("Enqueue error: %v", err)
		}
		jobIDs[i] = job.ID
	}

	// Claim all 10 jobs sequentially (simulating a single worker drain loop).
	claimed := 0
	seenIDs := make(map[string]bool)
	for {
		job, err := store.Claim(3)
		if err != nil {
			t.Fatalf("Claim error: %v", err)
		}
		if job == nil {
			break
		}
		if seenIDs[job.ID] {
			t.Errorf("job %s was claimed twice", job.ID)
		}
		seenIDs[job.ID] = true
		claimed++
	}

	if claimed != 10 {
		t.Errorf("expected 10 jobs claimed, got %d", claimed)
	}

	// Verify all jobs are running.
	var runningCount int64
	db.Model(&jobs.RefreshJob{}).Where("state = ?", jobs.JobStateRunning).Count(&runningCount)
	if runningCount != 10 {
		t.Errorf("expected 10 running jobs, got %d", runningCount)
	}

	// Verify each job was claimed exactly once.
	if len(seenIDs) != 10 {
		t.Errorf("expected 10 unique job IDs, got %d", len(seenIDs))
	}
}

// TestPhase8JobFailRetry verifies that a failed job is re-queued when retries remain.
func TestPhase8JobFailRetry(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	job := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	if _, err := store.Enqueue(job); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	// Claim, then fail with retries left.
	if _, err := store.Claim(3); err != nil {
		t.Fatalf("Claim error: %v", err)
	}
	if err := store.Fail(job.ID, "timeout", 3); err != nil {
		t.Fatalf("Fail error: %v", err)
	}

	result, err := store.Get(job.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.State != jobs.JobStateQueued {
		t.Errorf("expected re-queued state, got %q", result.State)
	}
	if result.LastError != "timeout" {
		t.Errorf("lastError = %q, want %q", result.LastError, "timeout")
	}
}

// TestPhase8JobCancelQueued verifies that a queued job can be canceled.
func TestPhase8JobCancelQueued(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	job := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	if _, err := store.Enqueue(job); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}

	if err := store.Cancel(job.ID); err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	result, err := store.Get(job.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.State != jobs.JobStateCanceled {
		t.Errorf("state = %q, want %q", result.State, jobs.JobStateCanceled)
	}
}

// TestPhase8JobAPIListHandler verifies the job list HTTP handler with filters.
func TestPhase8JobAPIListHandler(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	// Create jobs in different namespaces.
	for _, ns := range []string{"team-a", "team-a", "team-b"} {
		job := &jobs.RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      ns,
			Plugin:         "mcp",
			SourceID:       "src",
			RequestedBy:    "test",
			RequestedAt:    time.Now(),
			State:          jobs.JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		if _, err := store.Enqueue(job); err != nil {
			t.Fatalf("Enqueue error: %v", err)
		}
	}

	router := jobs.Router(store, nil)
	ts := httptest.NewServer(router)
	defer ts.Close()

	t.Run("list all jobs", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/refresh")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var result struct {
			Jobs      []map[string]any `json:"jobs"`
			TotalSize int              `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.TotalSize != 3 {
			t.Errorf("totalSize = %d, want 3", result.TotalSize)
		}
	})

	t.Run("filter by namespace", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/refresh?namespace=team-a")
		if err != nil {
			t.Fatalf("GET failed: %v", err)
		}
		defer resp.Body.Close()

		var result struct {
			TotalSize int `json:"totalSize"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if result.TotalSize != 2 {
			t.Errorf("totalSize = %d, want 2", result.TotalSize)
		}
	})
}

// TestPhase8RefreshReturns202 verifies that the refresh endpoint returns 202
// when a job store is available (async mode). This tests the handler pattern
// used by refreshAllHandler and refreshSourceHandler.
func TestPhase8RefreshReturns202(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	// Build a handler that simulates the async refresh pattern:
	// when a jobStore is available, enqueue a job and return 202.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		job := &jobs.RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "_all",
			RequestedBy:    "test-user",
			RequestedAt:    time.Now(),
			State:          jobs.JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}

		enqueued, err := store.Enqueue(job)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "queued",
			"jobId":  enqueued.ID,
		})
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/management/refresh", "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 202, got %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string `json:"status"`
		JobID  string `json:"jobId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if result.Status != "queued" {
		t.Errorf("status = %q, want %q", result.Status, "queued")
	}
	if result.JobID == "" {
		t.Error("jobId should not be empty")
	}

	// Verify the job was created in the store.
	job, err := store.Get(result.JobID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if job == nil {
		t.Fatal("job not found in store")
	}
	if job.State != jobs.JobStateQueued {
		t.Errorf("job state = %q, want %q", job.State, jobs.JobStateQueued)
	}
}

// TestPhase8RefreshReturns202LiveServer verifies refresh returns 202 against
// a live catalog server with async jobs enabled.
func TestPhase8RefreshReturns202LiveServer(t *testing.T) {
	if serverURL == "" {
		t.Skip("requires running catalog-server (set CATALOG_SERVER_URL)")
	}
	waitForReady(t)

	// Find a plugin with refresh capability.
	var response pluginsResponse
	getJSON(t, "/api/plugins", &response)

	for _, p := range response.Plugins {
		if p.Management == nil || !p.Management.Refresh {
			continue
		}

		t.Run(p.Name, func(t *testing.T) {
			refreshURL := serverURL + p.BasePath + "/management/refresh"
			req, _ := http.NewRequest(http.MethodPost, refreshURL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("POST %s failed: %v", refreshURL, err)
			}
			defer resp.Body.Close()

			// Accept 200 (sync), 202 (async), 403 (authz enabled), or 429 (rate limited).
			switch resp.StatusCode {
			case http.StatusOK:
				t.Log("refresh returned 200 (sync mode)")
			case http.StatusAccepted:
				var result struct {
					Status string `json:"status"`
					JobID  string `json:"jobId"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("decode error: %v", err)
				}
				if result.JobID == "" {
					t.Error("202 response should include jobId")
				}
				t.Logf("refresh returned 202 with jobId=%s", result.JobID)
			case http.StatusForbidden:
				t.Skip("refresh returned 403 (authz enabled, no credentials)")
			case http.StatusTooManyRequests:
				t.Skip("refresh returned 429 (rate limited)")
			default:
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected 200, 202, 403, or 429, got %d: %s", resp.StatusCode, string(body))
			}
		})
		return // Test one plugin.
	}

	t.Skip("no plugin with refresh capability found")
}

// TestPhase8NoDuplicateJobsUnderConcurrentEnqueue verifies that multiple
// enqueue attempts with the same idempotency key produce exactly one job.
// This simulates the multi-replica scenario where different replicas all
// try to enqueue the same refresh operation.
func TestPhase8NoDuplicateJobsUnderConcurrentEnqueue(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	idempKey := "default:mcp:_all:" + uuid.New().String()
	const replicaCount = 10

	// Simulate replicas sequentially (SQLite doesn't support concurrent
	// writers from goroutines on :memory:). This still validates the
	// idempotency logic that prevents duplicate jobs.
	results := make([]*jobs.RefreshJob, replicaCount)
	for i := 0; i < replicaCount; i++ {
		job := &jobs.RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "_all",
			RequestedBy:    "replica-" + uuid.New().String()[:8],
			RequestedAt:    time.Now(),
			State:          jobs.JobStateQueued,
			IdempotencyKey: idempKey,
		}
		var err error
		results[i], err = store.Enqueue(job)
		if err != nil {
			t.Fatalf("replica %d Enqueue error: %v", i, err)
		}
	}

	// All should return the same job ID (idempotency).
	firstID := results[0].ID
	for i := 1; i < replicaCount; i++ {
		if results[i].ID != firstID {
			t.Errorf("replica %d returned different job ID: %s vs %s", i, results[i].ID, firstID)
		}
	}

	// Verify exactly one queued job exists in the DB.
	var count int64
	db.Model(&jobs.RefreshJob{}).Where("state = ? AND idempotency_key = ?", jobs.JobStateQueued, idempKey).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 queued job with idempotency key, got %d", count)
	}
}

// TestPhase8JobTerminalStates verifies IsTerminal for various states.
func TestPhase8JobTerminalStates(t *testing.T) {
	tests := []struct {
		state    jobs.JobState
		terminal bool
	}{
		{jobs.JobStateQueued, false},
		{jobs.JobStateRunning, false},
		{jobs.JobStateSucceeded, true},
		{jobs.JobStateFailed, true},
		{jobs.JobStateCanceled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			job := &jobs.RefreshJob{State: tt.state}
			if job.IsTerminal() != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", job.IsTerminal(), tt.terminal)
			}
		})
	}
}

// TestPhase8JobStuckRecovery verifies that stuck running jobs are recovered.
func TestPhase8JobStuckRecovery(t *testing.T) {
	db := setupJobTestDB(t)
	store := jobs.NewJobStore(db)

	job := &jobs.RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test",
		RequestedAt:    time.Now(),
		State:          jobs.JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	if _, err := store.Enqueue(job); err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}
	if _, err := store.Claim(3); err != nil {
		t.Fatalf("Claim error: %v", err)
	}

	// Simulate stuck: set started_at far in the past.
	oldTime := time.Now().Add(-30 * time.Minute)
	db.Model(&jobs.RefreshJob{}).Where("id = ?", job.ID).Update("started_at", oldTime)

	recovered, err := store.CleanupStuckJobs(10 * time.Minute)
	if err != nil {
		t.Fatalf("CleanupStuckJobs error: %v", err)
	}
	if recovered != 1 {
		t.Errorf("expected 1 recovered, got %d", recovered)
	}

	result, err := store.Get(job.ID)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if result.State != jobs.JobStateQueued {
		t.Errorf("state = %q, want %q after recovery", result.State, jobs.JobStateQueued)
	}
}
