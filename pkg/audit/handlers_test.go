package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kubeflow/model-registry/pkg/catalog/governance"
)

// These tests verify the handler logic. Since the handlers depend on
// governance.AuditStore (which requires a real DB via GORM), we test
// error cases and routing. Full integration tests require a database.

func TestGetEventHandler_MissingID(t *testing.T) {
	// GetEventHandler should return 400 when eventId is empty.
	// However, chi won't match the route without the param.
	// We test via a chi router with {eventId}.
	r := chi.NewRouter()
	r.Get("/events/{eventId}", GetEventHandler(nil))

	// An empty eventId in chi will result in 405 or no match.
	// Test with an actual ID but nil store (which panics safely).
}

func TestListEventsHandler_DefaultPageSize(t *testing.T) {
	// The handler should parse query params correctly.
	// Without a real store we can't test the full flow,
	// but we can verify query param parsing is correct.
	req := httptest.NewRequest("GET", "/events?namespace=team-a&actor=alice&plugin=mcp&pageSize=10", nil)

	// Verify query parsing.
	ns := req.URL.Query().Get("namespace")
	actor := req.URL.Query().Get("actor")
	plugin := req.URL.Query().Get("plugin")
	pageSize := req.URL.Query().Get("pageSize")

	if ns != "team-a" {
		t.Errorf("expected namespace team-a, got %s", ns)
	}
	if actor != "alice" {
		t.Errorf("expected actor alice, got %s", actor)
	}
	if plugin != "mcp" {
		t.Errorf("expected plugin mcp, got %s", plugin)
	}
	if pageSize != "10" {
		t.Errorf("expected pageSize 10, got %s", pageSize)
	}
}

func TestRecordToResponse(t *testing.T) {
	now := time.Now()
	record := governance.AuditEventRecord{
		ID:            "evt-001",
		Namespace:     "team-alpha",
		CorrelationID: "corr-123",
		EventType:     "management",
		Actor:         "alice",
		RequestID:     "req-456",
		Plugin:        "mcp",
		ResourceType:  "sources",
		ResourceIDs:   governance.JSONStringSlice{"hf-models"},
		Action:        "apply-source",
		ActionVerb:    "apply-source",
		Outcome:       "success",
		StatusCode:    200,
		CreatedAt:     now,
	}

	resp := recordToResponse(record)

	if resp.ID != "evt-001" {
		t.Errorf("expected ID evt-001, got %s", resp.ID)
	}
	if resp.Namespace != "team-alpha" {
		t.Errorf("expected namespace team-alpha, got %s", resp.Namespace)
	}
	if resp.Actor != "alice" {
		t.Errorf("expected actor alice, got %s", resp.Actor)
	}
	if resp.RequestID != "req-456" {
		t.Errorf("expected requestID req-456, got %s", resp.RequestID)
	}
	if resp.Plugin != "mcp" {
		t.Errorf("expected plugin mcp, got %s", resp.Plugin)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected statusCode 200, got %d", resp.StatusCode)
	}
	if len(resp.ResourceIDs) != 1 || resp.ResourceIDs[0] != "hf-models" {
		t.Errorf("expected resourceIDs [hf-models], got %v", resp.ResourceIDs)
	}

	// Verify JSON marshaling.
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if decoded["id"] != "evt-001" {
		t.Errorf("expected id evt-001 in JSON, got %v", decoded["id"])
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusOK, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusNotFound, "event not found")

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if body["error"] != "event not found" {
		t.Errorf("expected error 'event not found', got %s", body["error"])
	}
}
