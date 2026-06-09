package http

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"status-incident/internal/domain"
)

// ============= SLA handler routes =============

func TestSLAHandlers_ReportsLifecycle(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "SLASys")

	// Generate a report (defaults applied for empty fields)
	w := f.do("POST", "/api/sla/reports", []byte(`{}`))
	if w.Code != http.StatusOK {
		t.Fatalf("generate report: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Generate with explicit fields
	w = f.do("POST", "/api/sla/reports", mustJSON(map[string]string{
		"title": "Weekly", "period": "weekly", "generated_by": "tester",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("generate report 2: expected 200, got %d", w.Code)
	}

	// Generate with invalid body -> 400
	w = f.do("POST", "/api/sla/reports", []byte(`not json`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("generate invalid: expected 400, got %d", w.Code)
	}

	// List reports
	w = f.do("GET", "/api/sla/reports?limit=5", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list reports: expected 200, got %d", w.Code)
	}

	// Get report 1
	w = f.do("GET", "/api/sla/reports/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get report: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Get report invalid ID
	w = f.do("GET", "/api/sla/reports/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get report invalid: expected 400, got %d", w.Code)
	}

	// Get report not found
	w = f.do("GET", "/api/sla/reports/9999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get report not found: expected 404, got %d", w.Code)
	}

	// Delete report 1
	w = f.do("DELETE", "/api/sla/reports/1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete report: expected 204, got %d", w.Code)
	}

	// Delete report invalid ID
	w = f.do("DELETE", "/api/sla/reports/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete report invalid: expected 400, got %d", w.Code)
	}
}

func TestSLAHandlers_Breaches(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "BreachSys")

	// List breaches (default)
	w := f.do("GET", "/api/sla/breaches?limit=10", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list breaches: expected 200, got %d", w.Code)
	}

	// List unacknowledged breaches
	w = f.do("GET", "/api/sla/breaches?unacknowledged=true", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list unacked: expected 200, got %d", w.Code)
	}

	// Check breaches
	w = f.do("POST", "/api/sla/breaches/check?period=weekly", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("check breaches: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Check breaches default period
	w = f.do("POST", "/api/sla/breaches/check", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("check breaches default: expected 200, got %d", w.Code)
	}

	// Acknowledge a non-existent breach (no body) -> service may still succeed/fail;
	// here it should at least return a JSON status or an error. We expect 200 on success.
	w = f.do("POST", "/api/sla/breaches/1/acknowledge", []byte(`{"acked_by":"me"}`))
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("ack breach: unexpected %d: %s", w.Code, w.Body.String())
	}

	// Acknowledge invalid ID
	w = f.do("POST", "/api/sla/breaches/abc/acknowledge", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ack invalid id: expected 400, got %d", w.Code)
	}

	// Acknowledge with no/invalid body defaults acked_by to system
	w = f.do("POST", "/api/sla/breaches/1/acknowledge", []byte(`bad json`))
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Fatalf("ack bad body: unexpected %d", w.Code)
	}
}

func TestSLAHandlers_SystemSLA(t *testing.T) {
	f := newFullServer(t, false)
	sys, _ := f.seedSystem(t, "SysSLA")

	// GetSystemSLA
	w := f.do("GET", "/api/systems/1/sla?period=monthly", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get system sla: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// GetSystemSLA invalid ID
	w = f.do("GET", "/api/systems/abc/sla", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get system sla invalid: expected 400, got %d", w.Code)
	}

	// GetSystemSLA not found -> 404
	w = f.do("GET", "/api/systems/9999/sla", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get system sla not found: expected 404, got %d", w.Code)
	}

	// UpdateSystemSLATarget valid
	w = f.do("PUT", "/api/systems/1/sla-target", []byte(`{"sla_target":99.95}`))
	if w.Code != http.StatusOK {
		t.Fatalf("update sla target: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// UpdateSystemSLATarget invalid ID
	w = f.do("PUT", "/api/systems/abc/sla-target", []byte(`{"sla_target":99.95}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update target invalid id: expected 400, got %d", w.Code)
	}

	// UpdateSystemSLATarget invalid body
	w = f.do("PUT", "/api/systems/1/sla-target", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update target bad body: expected 400, got %d", w.Code)
	}

	// UpdateSystemSLATarget out of range
	w = f.do("PUT", "/api/systems/1/sla-target", []byte(`{"sla_target":150}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update target out of range: expected 400, got %d", w.Code)
	}

	// UpdateSystemSLATarget not found system
	w = f.do("PUT", "/api/systems/9999/sla-target", []byte(`{"sla_target":99}`))
	if w.Code != http.StatusNotFound {
		t.Fatalf("update target not found: expected 404, got %d", w.Code)
	}

	// GetSystemBreaches
	w = f.do("GET", "/api/systems/1/sla/breaches?limit=10", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get system breaches: expected 200, got %d", w.Code)
	}

	// GetSystemBreaches invalid ID
	w = f.do("GET", "/api/systems/abc/sla/breaches", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get system breaches invalid: expected 400, got %d", w.Code)
	}
	_ = sys
}

// ============= Export / Import routes =============

func TestExportImport(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "ExportSys")

	// Export all
	w := f.do("GET", "/api/export", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("export all: expected 200, got %d", w.Code)
	}
	if cd := w.Header().Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Errorf("expected attachment content-disposition, got %q", cd)
	}
	exportBody := w.Body.Bytes()

	// Export logs only
	w = f.do("GET", "/api/export/logs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("export logs: expected 200, got %d", w.Code)
	}

	// Import the exported data back into a fresh server
	f2 := newFullServer(t, false)
	w = f2.do("POST", "/api/import", exportBody)
	if w.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "systems_imported") {
		t.Errorf("expected import result body, got %s", w.Body.String())
	}

	// Import invalid JSON -> 400
	w = f2.do("POST", "/api/import", []byte(`not json`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("import invalid: expected 400, got %d", w.Code)
	}
}

func TestImport_StatusesAndHeartbeat(t *testing.T) {
	f := newFullServer(t, false)

	now := time.Now()
	data := ExportData{
		ExportedAt: now,
		Version:    "1.0",
		Systems: []ExportSystem{
			{ID: 10, Name: "Imported Sys", Description: "d", URL: "https://x", Owner: "o", Status: "yellow", CreatedAt: now, UpdatedAt: now},
		},
		Dependencies: []ExportDependency{
			{ID: 20, SystemID: 10, Name: "Imported Dep", Status: "red", HeartbeatURL: "https://hb.example.com", HeartbeatInterval: 60, CreatedAt: now, UpdatedAt: now},
			// dependency referencing a non-imported system -> error path
			{ID: 21, SystemID: 999, Name: "Orphan Dep", Status: "green", CreatedAt: now, UpdatedAt: now},
		},
		Logs: []ExportLog{
			{ID: 30, SystemID: ptrInt64(10), OldStatus: "green", NewStatus: "yellow", Message: "m", Source: "manual", CreatedAt: now},
			// log referencing non-imported entities -> skipped
			{ID: 31, SystemID: ptrInt64(999), OldStatus: "green", NewStatus: "red", Source: "manual", CreatedAt: now},
			// log referencing imported dependency
			{ID: 32, DependencyID: ptrInt64(20), OldStatus: "green", NewStatus: "red", Source: "heartbeat", CreatedAt: now},
		},
	}

	w := f.do("POST", "/api/import", mustJSON(data))
	if w.Code != http.StatusOK {
		t.Fatalf("import: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result ImportResult
	if err := unmarshalBody(w, &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.SystemsImported != 1 {
		t.Errorf("expected 1 system imported, got %d", result.SystemsImported)
	}
	if result.DependenciesImported != 1 {
		t.Errorf("expected 1 dependency imported, got %d", result.DependenciesImported)
	}
	if len(result.Errors) == 0 {
		t.Errorf("expected at least one error for orphan dependency")
	}
}

// ============= Incident routes =============

func TestIncidentRoutes(t *testing.T) {
	f := newFullServer(t, false)
	f.seedSystem(t, "IncSys")

	// Create incident (default severity when empty)
	w := f.do("POST", "/api/incidents", mustJSON(incidentRequest{Title: "Boom", Message: "down"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("create incident: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var inc incidentResponse
	if err := unmarshalBody(w, &inc); err != nil {
		t.Fatalf("decode incident: %v", err)
	}

	// Create invalid body
	w = f.do("POST", "/api/incidents", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("create incident bad: expected 400, got %d", w.Code)
	}

	// Get incidents list
	w = f.do("GET", "/api/incidents?limit=10", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list incidents: expected 200, got %d", w.Code)
	}

	// Active incidents
	w = f.do("GET", "/api/incidents/active", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("active incidents: expected 200, got %d", w.Code)
	}

	// Recent incidents
	w = f.do("GET", "/api/incidents/recent?days=30", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("recent incidents: expected 200, got %d", w.Code)
	}

	// Get single incident
	w = f.do("GET", "/api/incidents/1", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get incident: expected 200, got %d", w.Code)
	}

	// Get incident invalid ID
	w = f.do("GET", "/api/incidents/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get incident invalid: expected 400, got %d", w.Code)
	}

	// Get incident not found
	w = f.do("GET", "/api/incidents/9999", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get incident not found: expected 404, got %d", w.Code)
	}

	// Acknowledge
	w = f.do("POST", "/api/incidents/1/acknowledge", []byte(`{"by":"oncall"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("ack incident: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Acknowledge invalid ID
	w = f.do("POST", "/api/incidents/abc/acknowledge", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("ack invalid: expected 400, got %d", w.Code)
	}

	// Update status
	w = f.do("POST", "/api/incidents/1/status", mustJSON(incidentStatusRequest{
		Status: string(domain.IncidentIdentified), Message: "found it", By: "eng",
	}))
	if w.Code != http.StatusOK {
		t.Fatalf("update incident status: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update status invalid ID
	w = f.do("POST", "/api/incidents/abc/status", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update status invalid id: expected 400, got %d", w.Code)
	}

	// Update status invalid body
	w = f.do("POST", "/api/incidents/1/status", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("update status bad body: expected 400, got %d", w.Code)
	}

	// Add update
	w = f.do("POST", "/api/incidents/1/updates", mustJSON(incidentUpdateRequest{Message: "progress"}))
	if w.Code != http.StatusCreated {
		t.Fatalf("add update: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Add update invalid ID
	w = f.do("POST", "/api/incidents/abc/updates", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("add update invalid id: expected 400, got %d", w.Code)
	}

	// Add update invalid body
	w = f.do("POST", "/api/incidents/1/updates", []byte(`bad`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("add update bad body: expected 400, got %d", w.Code)
	}

	// Get updates
	w = f.do("GET", "/api/incidents/1/updates", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get updates: expected 200, got %d", w.Code)
	}

	// Get updates invalid ID
	w = f.do("GET", "/api/incidents/abc/updates", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("get updates invalid: expected 400, got %d", w.Code)
	}

	// Resolve
	w = f.do("POST", "/api/incidents/1/resolve", []byte(`{"postmortem":"fixed","by":"eng"}`))
	if w.Code != http.StatusOK {
		t.Fatalf("resolve incident: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Resolve invalid ID
	w = f.do("POST", "/api/incidents/abc/resolve", []byte(`{}`))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("resolve invalid: expected 400, got %d", w.Code)
	}

	// Delete
	w = f.do("DELETE", "/api/incidents/1", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete incident: expected 204, got %d", w.Code)
	}

	// Delete invalid ID
	w = f.do("DELETE", "/api/incidents/abc", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("delete invalid: expected 400, got %d", w.Code)
	}
}

func ptrInt64(v int64) *int64 { return &v }
