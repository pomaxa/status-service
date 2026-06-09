package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"status-incident/internal/domain"
)

func requestCtx() context.Context { return context.Background() }

func mustSystem(t *testing.T, name string) *domain.System {
	t.Helper()
	s, err := domain.NewSystem(name, "d", "", "o")
	if err != nil {
		t.Fatalf("new system: %v", err)
	}
	return s
}

func mustDep(systemID int64) *domain.Dependency {
	return &domain.Dependency{SystemID: systemID, Name: "Dep", Status: domain.StatusGreen}
}

// reqWithID builds a request carrying a chi URL param value for the given key.
func reqWithID(method, target, key, val string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, val)
	return r.WithContext(withChi(r, rctx))
}

// TestAPIValidation_InvalidID covers the "invalid ID" 400 branches across
// handlers that parse an ID path parameter.
func TestAPIValidation_InvalidID(t *testing.T) {
	server, _, _ := setupTestServer()

	type call struct {
		name string
		fn   http.HandlerFunc
		key  string
	}
	calls := []call{
		{"getSystem", server.apiGetSystem, "id"},
		{"updateSystem", server.apiUpdateSystem, "id"},
		{"deleteSystem", server.apiDeleteSystem, "id"},
		{"updateSystemStatus", server.apiUpdateSystemStatus, "id"},
		{"getSystemLogs", server.apiGetSystemLogs, "id"},
		{"getSystemAnalytics", server.apiGetSystemAnalytics, "id"},
		{"getDependencies", server.apiGetDependencies, "systemId"},
		{"createDependency", server.apiCreateDependency, "systemId"},
		{"getDependency", server.apiGetDependency, "id"},
		{"updateDependency", server.apiUpdateDependency, "id"},
		{"deleteDependency", server.apiDeleteDependency, "id"},
		{"updateDependencyStatus", server.apiUpdateDependencyStatus, "id"},
		{"setHeartbeat", server.apiSetHeartbeat, "id"},
		{"clearHeartbeat", server.apiClearHeartbeat, "id"},
		{"getDependencyLogs", server.apiGetDependencyLogs, "id"},
		{"getDependencyAnalytics", server.apiGetDependencyAnalytics, "id"},
		{"getDependencyLatency", server.apiGetDependencyLatency, "id"},
		{"getDependencyUptime", server.apiGetDependencyUptime, "id"},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			r := reqWithID("GET", "/x", c.key, "not-a-number", []byte(`{}`))
			w := httptest.NewRecorder()
			c.fn(w, r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s invalid id: expected 400, got %d", c.name, w.Code)
			}
		})
	}
}

// TestAPIValidation_InvalidBody covers the "invalid request body" 400 branches
// for handlers that decode a JSON body (after a valid ID).
func TestAPIValidation_InvalidBody(t *testing.T) {
	server, systemRepo, depRepo := setupTestServer()
	ctx := requestCtx()

	// seed entities so the body-decode happens after a valid ID lookup
	sys := mustSystem(t, "S")
	systemRepo.Create(ctx, sys)
	depRepo.Create(ctx, mustDep(sys.ID))

	bad := []byte("not json")

	type call struct {
		name string
		fn   http.HandlerFunc
		key  string
		val  string
	}
	calls := []call{
		{"createSystem", server.apiCreateSystem, "", ""},
		{"updateSystem", server.apiUpdateSystem, "id", "1"},
		{"updateSystemStatus", server.apiUpdateSystemStatus, "id", "1"},
		{"createDependency", server.apiCreateDependency, "systemId", "1"},
		{"updateDependency", server.apiUpdateDependency, "id", "1"},
		{"updateDependencyStatus", server.apiUpdateDependencyStatus, "id", "1"},
		{"setHeartbeat", server.apiSetHeartbeat, "id", "1"},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			var r *http.Request
			if c.key == "" {
				r = httptest.NewRequest("POST", "/x", bytes.NewReader(bad))
				r.Header.Set("Content-Type", "application/json")
			} else {
				r = reqWithID("POST", "/x", c.key, c.val, bad)
			}
			w := httptest.NewRecorder()
			c.fn(w, r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s invalid body: expected 400, got %d: %s", c.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestAPIValidation_MaintenanceIncidentInvalidID covers invalid ID branches for
// maintenance and incident handlers (which use parseID under the hood).
func TestAPIValidation_MaintenanceIncidentInvalidID(t *testing.T) {
	server, _, _ := setupTestServer()

	handlers := []struct {
		name string
		fn   http.HandlerFunc
	}{
		{"getMaintenance", server.apiGetMaintenance},
		{"updateMaintenance", server.apiUpdateMaintenance},
		{"deleteMaintenance", server.apiDeleteMaintenance},
		{"cancelMaintenance", server.apiCancelMaintenance},
		{"getIncident", server.apiGetIncident},
		{"deleteIncident", server.apiDeleteIncident},
		{"acknowledgeIncident", server.apiAcknowledgeIncident},
		{"updateIncidentStatus", server.apiUpdateIncidentStatus},
		{"resolveIncident", server.apiResolveIncident},
		{"getIncidentUpdates", server.apiGetIncidentUpdates},
		{"addIncidentUpdate", server.apiAddIncidentUpdate},
	}

	for _, h := range handlers {
		t.Run(h.name, func(t *testing.T) {
			r := reqWithID("POST", "/x", "id", "bad", []byte(`{}`))
			w := httptest.NewRecorder()
			h.fn(w, r)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s invalid id: expected 400, got %d", h.name, w.Code)
			}
		})
	}
}
