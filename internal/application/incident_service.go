package application

import (
	"context"
	"fmt"

	"status-incident/internal/domain"
)

// IncidentService handles incident-related use cases
type IncidentService struct {
	incidentRepo domain.IncidentRepository
}

// NewIncidentService creates a new IncidentService
func NewIncidentService(incidentRepo domain.IncidentRepository) *IncidentService {
	return &IncidentService{
		incidentRepo: incidentRepo,
	}
}

// CreateIncident creates a new incident
func (s *IncidentService) CreateIncident(ctx context.Context, title, message string, severity domain.IncidentSeverity, systemIDs []int64) (*domain.Incident, error) {
	incident, err := domain.NewIncident(title, message, severity)
	if err != nil {
		return nil, fmt.Errorf("invalid incident data: %w", err)
	}

	if len(systemIDs) > 0 {
		incident.SetSystemIDs(systemIDs)
	}

	if err := s.incidentRepo.Create(ctx, incident); err != nil {
		return nil, fmt.Errorf("failed to create incident: %w", err)
	}

	// Create initial update entry
	update, _ := domain.NewIncidentUpdate(incident.ID, incident.Status, message, "system")
	if update != nil {
		s.incidentRepo.CreateUpdate(ctx, update)
	}

	return incident, nil
}

// GetIncident retrieves an incident by ID
func (s *IncidentService) GetIncident(ctx context.Context, id int64) (*domain.Incident, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	return incident, nil
}

// GetAllIncidents retrieves all incidents
func (s *IncidentService) GetAllIncidents(ctx context.Context, limit int) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetAll(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get incidents: %w", err)
	}
	return incidents, nil
}

// GetActiveIncidents retrieves all unresolved incidents
func (s *IncidentService) GetActiveIncidents(ctx context.Context) ([]*domain.Incident, error) {
	incidents, err := s.incidentRepo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active incidents: %w", err)
	}
	return incidents, nil
}

// GetRecentIncidents retrieves recently resolved incidents
func (s *IncidentService) GetRecentIncidents(ctx context.Context, days int) ([]*domain.Incident, error) {
	if days <= 0 {
		days = 7
	}
	incidents, err := s.incidentRepo.GetRecent(ctx, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent incidents: %w", err)
	}
	return incidents, nil
}

// GetIncidentUpdates retrieves timeline for an incident
func (s *IncidentService) GetIncidentUpdates(ctx context.Context, incidentID int64) ([]*domain.IncidentUpdate, error) {
	updates, err := s.incidentRepo.GetUpdates(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident updates: %w", err)
	}
	return updates, nil
}

// AcknowledgeIncident marks an incident as acknowledged
func (s *IncidentService) AcknowledgeIncident(ctx context.Context, id int64, by string) (*domain.Incident, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("incident not found: %d", id)
	}

	if err := incident.Acknowledge(by); err != nil {
		return nil, fmt.Errorf("failed to acknowledge: %w", err)
	}

	if err := s.incidentRepo.Update(ctx, incident); err != nil {
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	return incident, nil
}

// UpdateIncidentStatus updates the status of an incident
func (s *IncidentService) UpdateIncidentStatus(ctx context.Context, id int64, status domain.IncidentStatus, message, updatedBy string) (*domain.Incident, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("incident not found: %d", id)
	}

	if err := incident.UpdateStatus(status); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	if err := s.incidentRepo.Update(ctx, incident); err != nil {
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	// Add update to timeline
	update, _ := domain.NewIncidentUpdate(id, status, message, updatedBy)
	if update != nil {
		s.incidentRepo.CreateUpdate(ctx, update)
	}

	return incident, nil
}

// AddIncidentUpdate adds a timeline entry without changing status
func (s *IncidentService) AddIncidentUpdate(ctx context.Context, incidentID int64, message, createdBy string) (*domain.IncidentUpdate, error) {
	incident, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("incident not found: %d", incidentID)
	}

	update, err := domain.NewIncidentUpdate(incidentID, incident.Status, message, createdBy)
	if err != nil {
		return nil, fmt.Errorf("invalid update data: %w", err)
	}

	if err := s.incidentRepo.CreateUpdate(ctx, update); err != nil {
		return nil, fmt.Errorf("failed to create update: %w", err)
	}

	return update, nil
}

// ResolveIncident marks an incident as resolved
func (s *IncidentService) ResolveIncident(ctx context.Context, id int64, postmortem, resolvedBy string) (*domain.Incident, error) {
	incident, err := s.incidentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get incident: %w", err)
	}
	if incident == nil {
		return nil, fmt.Errorf("incident not found: %d", id)
	}

	if err := incident.Resolve(postmortem); err != nil {
		return nil, fmt.Errorf("failed to resolve: %w", err)
	}

	if err := s.incidentRepo.Update(ctx, incident); err != nil {
		return nil, fmt.Errorf("failed to update incident: %w", err)
	}

	// Add resolve update to timeline
	message := "Incident resolved"
	if postmortem != "" {
		message = "Incident resolved: " + postmortem
	}
	update, _ := domain.NewIncidentUpdate(id, domain.IncidentResolved, message, resolvedBy)
	if update != nil {
		s.incidentRepo.CreateUpdate(ctx, update)
	}

	return incident, nil
}

// DeleteIncident removes an incident
func (s *IncidentService) DeleteIncident(ctx context.Context, id int64) error {
	if err := s.incidentRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete incident: %w", err)
	}
	return nil
}
