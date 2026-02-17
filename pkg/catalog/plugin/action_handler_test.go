package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// actionTestPlugin implements CatalogPlugin and ActionProvider.
type actionTestPlugin struct {
	testMgmtPlugin
	actions       map[ActionScope][]ActionDefinition
	handleResult  *ActionResult
	handleErr     error
	lastRequest   *ActionRequest
	lastScope     ActionScope
	lastTargetID  string
}

func (p *actionTestPlugin) ListActions(scope ActionScope) []ActionDefinition {
	return p.actions[scope]
}

func (p *actionTestPlugin) HandleAction(_ context.Context, scope ActionScope, targetID string, req ActionRequest) (*ActionResult, error) {
	p.lastScope = scope
	p.lastTargetID = targetID
	p.lastRequest = &req
	return p.handleResult, p.handleErr
}

// Ensure interface compliance.
var _ ActionProvider = (*actionTestPlugin)(nil)
var _ CatalogPlugin = (*actionTestPlugin)(nil)

func TestActionHandler(t *testing.T) {
	t.Run("plugin without ActionProvider returns 501", func(t *testing.T) {
		p := &testMgmtPlugin{} // does not implement ActionProvider
		handler := actionHandler(p, ActionScopeSource)

		body := `{"action": "refresh"}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNotImplemented, rr.Code)
	})

	t.Run("missing action field returns 400", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {{ID: "refresh", DisplayName: "Refresh"}},
			},
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		body := `{"params": {}}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "action field is required")
	})

	t.Run("unknown action returns 400", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {{ID: "refresh", DisplayName: "Refresh"}},
			},
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		body := `{"action": "nonexistent"}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "unknown action")
	})

	t.Run("dry-run on action without support returns 400", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {{ID: "delete", DisplayName: "Delete", SupportsDryRun: false}},
			},
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		body := `{"action": "delete", "dryRun": true}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "does not support dry-run")
	})

	t.Run("valid source action dispatches correctly", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {
					{ID: "refresh", DisplayName: "Refresh", SupportsDryRun: true},
				},
			},
			handleResult: &ActionResult{
				Action:  "refresh",
				Status:  "completed",
				Message: "refreshed 5 entities",
			},
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		body := `{"action": "refresh", "params": {"force": true}}`
		req := httptest.NewRequest("POST", "/sources/my-source:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, ActionScopeSource, p.lastScope)
		assert.Equal(t, "my-source", p.lastTargetID)
		assert.Equal(t, "refresh", p.lastRequest.Action)

		var result ActionResult
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, "refresh", result.Action)
	})

	t.Run("valid asset action dispatches correctly", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeAsset: {
					{ID: "tag", DisplayName: "Tag", SupportsDryRun: true, Idempotent: true},
				},
			},
			handleResult: &ActionResult{
				Action: "tag",
				Status: "completed",
			},
		}

		r := chi.NewRouter()
		r.Post("/entities/{entityName}:action", actionHandler(p, ActionScopeAsset))

		body := `{"action": "tag", "params": {"tags": ["production", "v2"]}}`
		req := httptest.NewRequest("POST", "/entities/my-model:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, ActionScopeAsset, p.lastScope)
		assert.Equal(t, "my-model", p.lastTargetID)
	})

	t.Run("handler error returns 500", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {{ID: "refresh", DisplayName: "Refresh"}},
			},
			handleErr: fmt.Errorf("internal failure"),
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		body := `{"action": "refresh"}`
		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "internal failure")
	})

	t.Run("invalid JSON body returns 400", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeSource: {{ID: "refresh", DisplayName: "Refresh"}},
			},
		}

		r := chi.NewRouter()
		r.Post("/sources/{sourceId}:action", actionHandler(p, ActionScopeSource))

		req := httptest.NewRequest("POST", "/sources/src1:action", bytes.NewReader([]byte("not json")))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("dry-run with support dispatches correctly", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeAsset: {
					{ID: "tag", DisplayName: "Tag", SupportsDryRun: true},
				},
			},
			handleResult: &ActionResult{
				Action: "tag",
				Status: "dry-run",
			},
		}

		r := chi.NewRouter()
		r.Post("/entities/{entityName}:action", actionHandler(p, ActionScopeAsset))

		body := `{"action": "tag", "dryRun": true, "params": {"tags": ["test"]}}`
		req := httptest.NewRequest("POST", "/entities/my-model:action", bytes.NewReader([]byte(body)))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NotNil(t, p.lastRequest)
		assert.True(t, p.lastRequest.DryRun)
	})
}

func TestActionsListHandler(t *testing.T) {
	t.Run("plugin without ActionProvider returns empty list", func(t *testing.T) {
		p := &testMgmtPlugin{}
		handler := actionsListHandler(p, ActionScopeSource)

		req := httptest.NewRequest("GET", "/actions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(0), result["count"])
	})

	t.Run("plugin with actions returns list", func(t *testing.T) {
		p := &actionTestPlugin{
			actions: map[ActionScope][]ActionDefinition{
				ActionScopeAsset: {
					{ID: "tag", DisplayName: "Tag"},
					{ID: "deprecate", DisplayName: "Deprecate"},
				},
			},
		}
		handler := actionsListHandler(p, ActionScopeAsset)

		req := httptest.NewRequest("GET", "/actions", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		var result map[string]any
		err := json.Unmarshal(rr.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, float64(2), result["count"])
	})
}
