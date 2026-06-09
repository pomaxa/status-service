package application

import (
	"context"
	"errors"
	"status-incident/internal/domain"
	"testing"
	"time"
)

// failingMaintenanceRepo returns errors from every method (or a fixed GetByID value)
// to exercise wrapped-error branches.
type failingMaintenanceRepo struct {
	getByID *domain.Maintenance
}

func (m *failingMaintenanceRepo) Create(ctx context.Context, maint *domain.Maintenance) error {
	return errors.New("create error")
}
func (m *failingMaintenanceRepo) GetByID(ctx context.Context, id int64) (*domain.Maintenance, error) {
	if m.getByID != nil {
		return m.getByID, nil
	}
	return nil, errors.New("get error")
}
func (m *failingMaintenanceRepo) GetAll(ctx context.Context) ([]*domain.Maintenance, error) {
	return nil, errors.New("getall error")
}
func (m *failingMaintenanceRepo) GetActive(ctx context.Context) ([]*domain.Maintenance, error) {
	return nil, errors.New("getactive error")
}
func (m *failingMaintenanceRepo) GetUpcoming(ctx context.Context) ([]*domain.Maintenance, error) {
	return nil, errors.New("getupcoming error")
}
func (m *failingMaintenanceRepo) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.Maintenance, error) {
	return nil, errors.New("getbytimerange error")
}
func (m *failingMaintenanceRepo) Update(ctx context.Context, maint *domain.Maintenance) error {
	return errors.New("update error")
}
func (m *failingMaintenanceRepo) Delete(ctx context.Context, id int64) error {
	return errors.New("delete error")
}

// ============= Maintenance Service error-path coverage =============

func TestMaintenanceService_CreateMaintenance_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	if _, err := service.CreateMaintenance(context.Background(), "Title", "Desc", start, end, nil); err == nil {
		t.Error("expected create error")
	}
}

func TestMaintenanceService_GetMaintenance_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, err := service.GetMaintenance(context.Background(), 1); err == nil {
		t.Error("expected get error")
	}
}

func TestMaintenanceService_GetAllMaintenances_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, err := service.GetAllMaintenances(context.Background()); err == nil {
		t.Error("expected getall error")
	}
}

func TestMaintenanceService_GetActiveMaintenances_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, err := service.GetActiveMaintenances(context.Background()); err == nil {
		t.Error("expected getactive error")
	}
}

func TestMaintenanceService_GetUpcomingMaintenances_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, err := service.GetUpcomingMaintenances(context.Background()); err == nil {
		t.Error("expected getupcoming error")
	}
}

func TestMaintenanceService_UpdateMaintenance_GetError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	if _, err := service.UpdateMaintenance(context.Background(), 1, "Title", "Desc", start, end, nil); err == nil {
		t.Error("expected get error")
	}
}

func TestMaintenanceService_UpdateMaintenance_InvalidData(t *testing.T) {
	maint, _ := domain.NewMaintenance("Old", "", time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	maint.ID = 1
	service := NewMaintenanceService(&failingMaintenanceRepo{getByID: maint})

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	// Empty title -> Update validation error
	if _, err := service.UpdateMaintenance(context.Background(), 1, "", "Desc", start, end, nil); err == nil {
		t.Error("expected invalid update data error")
	}
}

func TestMaintenanceService_UpdateMaintenance_SaveError(t *testing.T) {
	maint, _ := domain.NewMaintenance("Old", "", time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	maint.ID = 1
	service := NewMaintenanceService(&failingMaintenanceRepo{getByID: maint})

	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	// Valid update, but repo Update fails
	if _, err := service.UpdateMaintenance(context.Background(), 1, "New", "Desc", start, end, []int64{1}); err == nil {
		t.Error("expected save error")
	}
}

func TestMaintenanceService_CancelMaintenance_GetError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, err := service.CancelMaintenance(context.Background(), 1); err == nil {
		t.Error("expected get error")
	}
}

func TestMaintenanceService_CancelMaintenance_SaveError(t *testing.T) {
	maint, _ := domain.NewMaintenance("Old", "", time.Now().Add(time.Hour), time.Now().Add(2*time.Hour))
	maint.ID = 1
	service := NewMaintenanceService(&failingMaintenanceRepo{getByID: maint})

	if _, err := service.CancelMaintenance(context.Background(), 1); err == nil {
		t.Error("expected save error")
	}
}

func TestMaintenanceService_DeleteMaintenance_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if err := service.DeleteMaintenance(context.Background(), 1); err == nil {
		t.Error("expected delete error")
	}
}

func TestMaintenanceService_IsSystemUnderMaintenance_RepoError(t *testing.T) {
	service := NewMaintenanceService(&failingMaintenanceRepo{})
	if _, _, err := service.IsSystemUnderMaintenance(context.Background(), 1); err == nil {
		t.Error("expected getactive error")
	}
}

func TestMaintenanceService_IsSystemUnderMaintenance_Match(t *testing.T) {
	maintRepo := NewMockMaintenanceRepository()
	// Active maintenance affecting system 1
	now := time.Now()
	maint, _ := domain.NewMaintenance("Active", "", now.Add(-time.Hour), now.Add(time.Hour))
	maint.SetSystemIDs([]int64{1})
	maintRepo.Create(context.Background(), maint)

	service := NewMaintenanceService(maintRepo)

	under, found, err := service.IsSystemUnderMaintenance(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !under || found == nil {
		t.Error("expected system to be under maintenance")
	}

	// System not affected
	under2, _, _ := service.IsSystemUnderMaintenance(context.Background(), 999)
	if under2 {
		t.Error("expected system 999 not under maintenance")
	}
}
