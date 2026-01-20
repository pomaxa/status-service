package application

import (
	"context"
	"fmt"
	"time"

	"status-incident/internal/domain"
)

// MaintenanceService handles maintenance-related use cases
type MaintenanceService struct {
	maintenanceRepo domain.MaintenanceRepository
}

// NewMaintenanceService creates a new MaintenanceService
func NewMaintenanceService(maintenanceRepo domain.MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{
		maintenanceRepo: maintenanceRepo,
	}
}

// CreateMaintenance creates a new maintenance window
func (s *MaintenanceService) CreateMaintenance(ctx context.Context, title, description string, startTime, endTime time.Time, systemIDs []int64) (*domain.Maintenance, error) {
	m, err := domain.NewMaintenance(title, description, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("invalid maintenance data: %w", err)
	}

	if len(systemIDs) > 0 {
		m.SetSystemIDs(systemIDs)
	}

	if err := s.maintenanceRepo.Create(ctx, m); err != nil {
		return nil, fmt.Errorf("failed to create maintenance: %w", err)
	}

	return m, nil
}

// GetMaintenance retrieves a maintenance window by ID
func (s *MaintenanceService) GetMaintenance(ctx context.Context, id int64) (*domain.Maintenance, error) {
	m, err := s.maintenanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance: %w", err)
	}
	return m, nil
}

// GetAllMaintenances retrieves all maintenance windows
func (s *MaintenanceService) GetAllMaintenances(ctx context.Context) ([]*domain.Maintenance, error) {
	maintenances, err := s.maintenanceRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenances: %w", err)
	}
	return maintenances, nil
}

// GetActiveMaintenances retrieves currently active maintenance windows
func (s *MaintenanceService) GetActiveMaintenances(ctx context.Context) ([]*domain.Maintenance, error) {
	maintenances, err := s.maintenanceRepo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active maintenances: %w", err)
	}
	return maintenances, nil
}

// GetUpcomingMaintenances retrieves scheduled maintenance windows
func (s *MaintenanceService) GetUpcomingMaintenances(ctx context.Context) ([]*domain.Maintenance, error) {
	maintenances, err := s.maintenanceRepo.GetUpcoming(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming maintenances: %w", err)
	}
	return maintenances, nil
}

// UpdateMaintenance updates a maintenance window
func (s *MaintenanceService) UpdateMaintenance(ctx context.Context, id int64, title, description string, startTime, endTime time.Time, systemIDs []int64) (*domain.Maintenance, error) {
	m, err := s.maintenanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("maintenance not found: %d", id)
	}

	if err := m.Update(title, description, startTime, endTime); err != nil {
		return nil, fmt.Errorf("invalid update data: %w", err)
	}

	m.SetSystemIDs(systemIDs)

	if err := s.maintenanceRepo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("failed to update maintenance: %w", err)
	}

	return m, nil
}

// CancelMaintenance cancels a maintenance window
func (s *MaintenanceService) CancelMaintenance(ctx context.Context, id int64) (*domain.Maintenance, error) {
	m, err := s.maintenanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("maintenance not found: %d", id)
	}

	m.Cancel()

	if err := s.maintenanceRepo.Update(ctx, m); err != nil {
		return nil, fmt.Errorf("failed to cancel maintenance: %w", err)
	}

	return m, nil
}

// DeleteMaintenance removes a maintenance window
func (s *MaintenanceService) DeleteMaintenance(ctx context.Context, id int64) error {
	if err := s.maintenanceRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete maintenance: %w", err)
	}
	return nil
}

// IsSystemUnderMaintenance checks if a system is currently under maintenance
func (s *MaintenanceService) IsSystemUnderMaintenance(ctx context.Context, systemID int64) (bool, *domain.Maintenance, error) {
	actives, err := s.maintenanceRepo.GetActive(ctx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to check maintenance: %w", err)
	}

	for _, m := range actives {
		if m.AffectsSystem(systemID) {
			return true, m, nil
		}
	}

	return false, nil, nil
}
