package jobs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RefreshJob{}))
	return db
}

func setupRouter(store *JobStore) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/refresh/{jobId}", GetJobHandler(store))
	r.Get("/refresh", ListJobsHandler(store))
	r.Post("/refresh/{jobId}:cancel", CancelJobHandler(store))
	return r
}

func TestGetJobHandler_Found(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		SourceID:       "src1",
		RequestedBy:    "test-user",
		RequestedAt:    time.Now().Truncate(time.Second),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/refresh/"+job.ID, nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp jobResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, job.ID, resp.ID)
	assert.Equal(t, "queued", resp.State)
	assert.Equal(t, "mcp", resp.Plugin)
	assert.Equal(t, "test-user", resp.RequestedBy)
}

func TestGetJobHandler_NotFound(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/refresh/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListJobsHandler_Pagination(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	// Create 3 jobs.
	for i := 0; i < 3; i++ {
		job := &RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         "mcp",
			SourceID:       "src",
			RequestedBy:    "user",
			RequestedAt:    time.Now().Add(time.Duration(i) * time.Minute),
			State:          JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		_, err := store.Enqueue(job)
		require.NoError(t, err)
	}

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/refresh?pageSize=2", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	jobs := resp["jobs"].([]any)
	assert.Len(t, jobs, 2)
	assert.NotEmpty(t, resp["nextPageToken"])
	assert.Equal(t, float64(3), resp["totalSize"])
}

func TestListJobsHandler_FilterByPlugin(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	for _, plugin := range []string{"mcp", "knowledge"} {
		job := &RefreshJob{
			ID:             uuid.New().String(),
			Namespace:      "default",
			Plugin:         plugin,
			RequestedBy:    "user",
			RequestedAt:    time.Now(),
			State:          JobStateQueued,
			IdempotencyKey: uuid.New().String(),
		}
		_, err := store.Enqueue(job)
		require.NoError(t, err)
	}

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodGet, "/refresh?plugin=mcp", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))

	jobs := resp["jobs"].([]any)
	assert.Len(t, jobs, 1)
	assert.Equal(t, float64(1), resp["totalSize"])
}

func TestCancelJobHandler_QueuedJob(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		RequestedBy:    "user",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodPost, "/refresh/"+job.ID+":cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "canceled", resp["status"])
}

func TestCancelJobHandler_RunningJobFails(t *testing.T) {
	db := setupHandlerTestDB(t)
	store := NewJobStore(db)

	job := &RefreshJob{
		ID:             uuid.New().String(),
		Namespace:      "default",
		Plugin:         "mcp",
		RequestedBy:    "user",
		RequestedAt:    time.Now(),
		State:          JobStateQueued,
		IdempotencyKey: uuid.New().String(),
	}
	_, err := store.Enqueue(job)
	require.NoError(t, err)

	// Transition to running.
	_, err = store.Claim(3)
	require.NoError(t, err)

	r := setupRouter(store)
	req := httptest.NewRequest(http.MethodPost, "/refresh/"+job.ID+":cancel", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	// The cancel handler returns 400 for non-cancelable jobs; we could also use 409.
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusConflict,
		"expected 400 or 409, got %d", w.Code)
}
