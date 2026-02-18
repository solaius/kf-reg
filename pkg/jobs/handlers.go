package jobs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// GetJobHandler handles GET /api/jobs/v1alpha1/refresh/{jobId}
func GetJobHandler(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobId")
		if jobID == "" {
			writeError(w, http.StatusBadRequest, "missing job ID")
			return
		}

		job, err := store.Get(jobID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get job: %v", err))
			return
		}
		if job == nil {
			writeError(w, http.StatusNotFound, fmt.Sprintf("job %q not found", jobID))
			return
		}

		writeJSON(w, http.StatusOK, jobToResponse(job))
	}
}

// ListJobsHandler handles GET /api/jobs/v1alpha1/refresh
// Query params: namespace, plugin, sourceId, state, requestedBy, pageSize, pageToken
func ListJobsHandler(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filter := JobListFilter{
			Namespace:   r.URL.Query().Get("namespace"),
			Plugin:      r.URL.Query().Get("plugin"),
			SourceID:    r.URL.Query().Get("sourceId"),
			State:       r.URL.Query().Get("state"),
			RequestedBy: r.URL.Query().Get("requestedBy"),
		}

		pageSize := 20
		if ps := r.URL.Query().Get("pageSize"); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 {
				pageSize = v
			}
		}
		pageToken := r.URL.Query().Get("pageToken")

		records, nextToken, total, err := store.List(filter, pageSize, pageToken)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list jobs: %v", err))
			return
		}

		jobs := make([]jobResponse, len(records))
		for i := range records {
			jobs[i] = jobToResponse(&records[i])
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"jobs":          jobs,
			"nextPageToken": nextToken,
			"totalSize":     total,
		})
	}
}

// CancelJobHandler handles POST /api/jobs/v1alpha1/refresh/{jobId}:cancel
func CancelJobHandler(store *JobStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := chi.URLParam(r, "jobId")
		if jobID == "" {
			writeError(w, http.StatusBadRequest, "missing job ID")
			return
		}

		if err := store.Cancel(jobID); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to cancel job: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "canceled",
			"jobId":  jobID,
		})
	}
}

// jobResponse is the API response for a refresh job.
type jobResponse struct {
	ID              string `json:"id"`
	Namespace       string `json:"namespace"`
	Plugin          string `json:"plugin"`
	SourceID        string `json:"sourceId,omitempty"`
	RequestedBy     string `json:"requestedBy"`
	RequestedAt     string `json:"requestedAt"`
	State           string `json:"state"`
	Progress        string `json:"progress,omitempty"`
	Message         string `json:"message,omitempty"`
	StartedAt       string `json:"startedAt,omitempty"`
	FinishedAt      string `json:"finishedAt,omitempty"`
	AttemptCount    int    `json:"attemptCount"`
	LastError       string `json:"lastError,omitempty"`
	EntitiesLoaded  int    `json:"entitiesLoaded,omitempty"`
	EntitiesRemoved int    `json:"entitiesRemoved,omitempty"`
	DurationMs      int64  `json:"durationMs,omitempty"`
}

func jobToResponse(job *RefreshJob) jobResponse {
	resp := jobResponse{
		ID:              job.ID,
		Namespace:       job.Namespace,
		Plugin:          job.Plugin,
		SourceID:        job.SourceID,
		RequestedBy:     job.RequestedBy,
		RequestedAt:     job.RequestedAt.Format(time.RFC3339),
		State:           string(job.State),
		Progress:        job.Progress,
		Message:         job.Message,
		AttemptCount:    job.AttemptCount,
		LastError:       job.LastError,
		EntitiesLoaded:  job.EntitiesLoaded,
		EntitiesRemoved: job.EntitiesRemoved,
		DurationMs:      job.DurationMs,
	}
	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt.Format(time.RFC3339)
	}
	if job.FinishedAt != nil {
		resp.FinishedAt = job.FinishedAt.Format(time.RFC3339)
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
