package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
)

// failingIncidentRepo lets every method return an error to cover wrapped-error branches.
type failingIncidentRepo struct {
	getByID *domain.Incident
}

func (m *failingIncidentRepo) Create(ctx context.Context, i *domain.Incident) error {
	return errors.New("create error")
}
func (m *failingIncidentRepo) GetByID(ctx context.Context, id int64) (*domain.Incident, error) {
	if m.getByID != nil {
		return m.getByID, nil
	}
	return nil, errors.New("get error")
}
func (m *failingIncidentRepo) GetAll(ctx context.Context, limit int) ([]*domain.Incident, error) {
	return nil, errors.New("getall error")
}
func (m *failingIncidentRepo) GetActive(ctx context.Context) ([]*domain.Incident, error) {
	return nil, errors.New("getactive error")
}
func (m *failingIncidentRepo) GetRecent(ctx context.Context, days int) ([]*domain.Incident, error) {
	return nil, errors.New("getrecent error")
}
func (m *failingIncidentRepo) Update(ctx context.Context, i *domain.Incident) error {
	return errors.New("update error")
}
func (m *failingIncidentRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("delete error")
}
func (m *failingIncidentRepo) CreateUpdate(ctx context.Context, u *domain.IncidentUpdate) error {
	return errors.New("createupdate error")
}
func (m *failingIncidentRepo) GetUpdates(ctx context.Context, incidentID int64) ([]*domain.IncidentUpdate, error) {
	return nil, errors.New("getupdates error")
}

// ============= Incident Service error-path coverage =============

func TestIncidentService_CreateIncident_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	_, err := service.CreateIncident(context.Background(), "Title", "Message", domain.SeverityMinor, nil)
	if err == nil {
		t.Error("expected create error")
	}
}

func TestIncidentService_GetIncident_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.GetIncident(context.Background(), 1); err == nil {
		t.Error("expected get error")
	}
}

func TestIncidentService_GetAllIncidents_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.GetAllIncidents(context.Background(), 10); err == nil {
		t.Error("expected getall error")
	}
}

func TestIncidentService_GetActiveIncidents_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.GetActiveIncidents(context.Background()); err == nil {
		t.Error("expected getactive error")
	}
}

func TestIncidentService_GetRecentIncidents_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.GetRecentIncidents(context.Background(), 0); err == nil {
		t.Error("expected getrecent error")
	}
}

func TestIncidentService_GetIncidentUpdates_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.GetIncidentUpdates(context.Background(), 1); err == nil {
		t.Error("expected getupdates error")
	}
}

func TestIncidentService_AcknowledgeIncident_GetError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.AcknowledgeIncident(context.Background(), 1, "ops"); err == nil {
		t.Error("expected get error")
	}
}

func TestIncidentService_AcknowledgeIncident_AlreadyAcked(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	incident.Acknowledge("first")
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	// Already acknowledged -> Acknowledge() returns error
	if _, err := service.AcknowledgeIncident(context.Background(), 1, "second"); err == nil {
		t.Error("expected already-acknowledged error")
	}
}

func TestIncidentService_AcknowledgeIncident_UpdateError(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	if _, err := service.AcknowledgeIncident(context.Background(), 1, "ops"); err == nil {
		t.Error("expected update error")
	}
}

func TestIncidentService_UpdateIncidentStatus_GetError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.UpdateIncidentStatus(context.Background(), 1, domain.IncidentIdentified, "msg", "ops"); err == nil {
		t.Error("expected get error")
	}
}

func TestIncidentService_UpdateIncidentStatus_StatusError(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	incident.Resolve("done") // resolved -> UpdateStatus returns error
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	if _, err := service.UpdateIncidentStatus(context.Background(), 1, domain.IncidentIdentified, "msg", "ops"); err == nil {
		t.Error("expected update status error")
	}
}

func TestIncidentService_UpdateIncidentStatus_UpdateError(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	if _, err := service.UpdateIncidentStatus(context.Background(), 1, domain.IncidentIdentified, "msg", "ops"); err == nil {
		t.Error("expected repo update error")
	}
}

func TestIncidentService_AddIncidentUpdate_GetError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.AddIncidentUpdate(context.Background(), 1, "msg", "ops"); err == nil {
		t.Error("expected get error")
	}
}

func TestIncidentService_AddIncidentUpdate_InvalidUpdate(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	// Empty message -> NewIncidentUpdate returns error
	if _, err := service.AddIncidentUpdate(context.Background(), 1, "", "ops"); err == nil {
		t.Error("expected invalid update data error")
	}
}

func TestIncidentService_AddIncidentUpdate_CreateError(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	// Valid message but CreateUpdate fails
	if _, err := service.AddIncidentUpdate(context.Background(), 1, "real message", "ops"); err == nil {
		t.Error("expected create update error")
	}
}

func TestIncidentService_ResolveIncident_GetError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if _, err := service.ResolveIncident(context.Background(), 1, "pm", "ops"); err == nil {
		t.Error("expected get error")
	}
}

func TestIncidentService_ResolveIncident_AlreadyResolved(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	incident.Resolve("first")
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	if _, err := service.ResolveIncident(context.Background(), 1, "again", "ops"); err == nil {
		t.Error("expected already-resolved error")
	}
}

func TestIncidentService_ResolveIncident_UpdateError(t *testing.T) {
	incident, _ := domain.NewIncident("Title", "Message", domain.SeverityMinor)
	incident.ID = 1
	service := NewIncidentService(&failingIncidentRepo{getByID: incident})

	if _, err := service.ResolveIncident(context.Background(), 1, "pm", "ops"); err == nil {
		t.Error("expected repo update error")
	}
}

func TestIncidentService_DeleteIncident_RepoError(t *testing.T) {
	service := NewIncidentService(&failingIncidentRepo{})
	if err := service.DeleteIncident(context.Background(), 1); err == nil {
		t.Error("expected delete error")
	}
}

// CreateIncident with a valid initial update but failing CreateUpdate is non-fatal:
// CreateIncident swallows the CreateUpdate result, so the incident is returned.
func TestIncidentService_CreateIncident_UpdateNonFatal(t *testing.T) {
	incRepo := NewMockIncidentRepository()
	service := NewIncidentService(incRepo)

	incident, err := service.CreateIncident(context.Background(), "Title", "Message", domain.SeverityMajor, []int64{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if incident == nil {
		t.Fatal("expected non-nil incident")
	}
	if len(incident.SystemIDs) != 2 {
		t.Errorf("expected 2 system IDs, got %d", len(incident.SystemIDs))
	}
}
