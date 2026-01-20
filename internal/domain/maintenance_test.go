package domain

import (
	"testing"
	"time"
)

func TestNewMaintenance(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(2 * time.Hour)
	past := now.Add(-2 * time.Hour)
	recentPast := now.Add(-1 * time.Hour)

	tests := []struct {
		name          string
		title         string
		description   string
		startTime     time.Time
		endTime       time.Time
		wantErr       bool
		errContains   string
		expectedStatus MaintenanceStatus
	}{
		{
			name:          "valid future maintenance",
			title:         "Scheduled Maintenance",
			description:   "Database upgrade",
			startTime:     future,
			endTime:       farFuture,
			wantErr:       false,
			expectedStatus: MaintenanceScheduled,
		},
		{
			name:          "valid current maintenance",
			title:         "Current Maintenance",
			description:   "In progress",
			startTime:     recentPast,
			endTime:       future,
			wantErr:       false,
			expectedStatus: MaintenanceInProgress,
		},
		{
			name:          "valid past maintenance",
			title:         "Past Maintenance",
			description:   "Completed",
			startTime:     past,
			endTime:       recentPast,
			wantErr:       false,
			expectedStatus: MaintenanceCompleted,
		},
		{
			name:        "empty title",
			title:       "",
			description: "Description",
			startTime:   future,
			endTime:     farFuture,
			wantErr:     true,
			errContains: "title is required",
		},
		{
			name:        "zero start time",
			title:       "Maintenance",
			description: "Description",
			startTime:   time.Time{},
			endTime:     farFuture,
			wantErr:     true,
			errContains: "start time is required",
		},
		{
			name:        "zero end time",
			title:       "Maintenance",
			description: "Description",
			startTime:   future,
			endTime:     time.Time{},
			wantErr:     true,
			errContains: "end time is required",
		},
		{
			name:        "end before start",
			title:       "Maintenance",
			description: "Description",
			startTime:   farFuture,
			endTime:     future,
			wantErr:     true,
			errContains: "end time must be after start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMaintenance(tt.title, tt.description, tt.startTime, tt.endTime)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if m.Title != tt.title {
				t.Errorf("expected title %q, got %q", tt.title, m.Title)
			}
			if m.Description != tt.description {
				t.Errorf("expected description %q, got %q", tt.description, m.Description)
			}
			if m.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, m.Status)
			}
			if m.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}
		})
	}
}

func TestMaintenance_Update(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	farFuture := now.Add(2 * time.Hour)

	m, _ := NewMaintenance("Original", "Original desc", future, farFuture)

	tests := []struct {
		name        string
		newTitle    string
		newDesc     string
		newStart    time.Time
		newEnd      time.Time
		wantErr     bool
	}{
		{
			name:     "valid update",
			newTitle: "Updated",
			newDesc:  "Updated desc",
			newStart: future,
			newEnd:   farFuture,
			wantErr:  false,
		},
		{
			name:     "empty title",
			newTitle: "",
			newDesc:  "Desc",
			newStart: future,
			newEnd:   farFuture,
			wantErr:  true,
		},
		{
			name:     "end before start",
			newTitle: "Title",
			newDesc:  "Desc",
			newStart: farFuture,
			newEnd:   future,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.Update(tt.newTitle, tt.newDesc, tt.newStart, tt.newEnd)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if m.Title != tt.newTitle {
				t.Errorf("expected title %q, got %q", tt.newTitle, m.Title)
			}
		})
	}
}

func TestMaintenance_SetSystemIDs(t *testing.T) {
	now := time.Now()
	m, _ := NewMaintenance("Test", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))
	oldUpdatedAt := m.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	m.SetSystemIDs([]int64{1, 2, 3})

	if len(m.SystemIDs) != 3 {
		t.Errorf("expected 3 system IDs, got %d", len(m.SystemIDs))
	}
	if !m.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestMaintenance_RefreshStatus(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name           string
		startTime      time.Time
		endTime        time.Time
		initialStatus  MaintenanceStatus
		expectedStatus MaintenanceStatus
	}{
		{
			name:           "future becomes scheduled",
			startTime:      now.Add(1 * time.Hour),
			endTime:        now.Add(2 * time.Hour),
			initialStatus:  MaintenanceScheduled,
			expectedStatus: MaintenanceScheduled,
		},
		{
			name:           "current becomes in_progress",
			startTime:      now.Add(-1 * time.Hour),
			endTime:        now.Add(1 * time.Hour),
			initialStatus:  MaintenanceScheduled,
			expectedStatus: MaintenanceInProgress,
		},
		{
			name:           "past becomes completed",
			startTime:      now.Add(-2 * time.Hour),
			endTime:        now.Add(-1 * time.Hour),
			initialStatus:  MaintenanceScheduled,
			expectedStatus: MaintenanceCompleted,
		},
		{
			name:           "cancelled stays cancelled",
			startTime:      now.Add(-1 * time.Hour),
			endTime:        now.Add(1 * time.Hour),
			initialStatus:  MaintenanceCancelled,
			expectedStatus: MaintenanceCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Maintenance{
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
				Status:    tt.initialStatus,
			}

			m.RefreshStatus()

			if m.Status != tt.expectedStatus {
				t.Errorf("expected status %q, got %q", tt.expectedStatus, m.Status)
			}
		})
	}
}

func TestMaintenance_Cancel(t *testing.T) {
	now := time.Now()
	m, _ := NewMaintenance("Test", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))

	m.Cancel()

	if m.Status != MaintenanceCancelled {
		t.Errorf("expected status cancelled, got %q", m.Status)
	}
}

func TestMaintenance_IsActive(t *testing.T) {
	now := time.Now()

	// Future maintenance - not active
	m1, _ := NewMaintenance("Future", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))
	if m1.IsActive() {
		t.Error("expected future maintenance to not be active")
	}

	// Current maintenance - active
	m2, _ := NewMaintenance("Current", "Desc", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if !m2.IsActive() {
		t.Error("expected current maintenance to be active")
	}

	// Past maintenance - not active
	m3, _ := NewMaintenance("Past", "Desc", now.Add(-2*time.Hour), now.Add(-1*time.Hour))
	if m3.IsActive() {
		t.Error("expected past maintenance to not be active")
	}
}

func TestMaintenance_AffectsSystem(t *testing.T) {
	now := time.Now()
	m, _ := NewMaintenance("Test", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))

	tests := []struct {
		name      string
		systemIDs []int64
		checkID   int64
		expected  bool
	}{
		{"nil affects all", nil, 1, true},
		{"empty affects all", []int64{}, 1, true},
		{"specific system match", []int64{1, 2, 3}, 2, true},
		{"specific system no match", []int64{1, 2, 3}, 5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.SystemIDs = tt.systemIDs
			result := m.AffectsSystem(tt.checkID)
			if result != tt.expected {
				t.Errorf("AffectsSystem(%d) = %v, want %v", tt.checkID, result, tt.expected)
			}
		})
	}
}

func TestMaintenance_IsUpcoming(t *testing.T) {
	now := time.Now()

	// Future scheduled maintenance - upcoming
	m1, _ := NewMaintenance("Future", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))
	if !m1.IsUpcoming() {
		t.Error("expected future maintenance to be upcoming")
	}

	// Current maintenance - not upcoming
	m2, _ := NewMaintenance("Current", "Desc", now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if m2.IsUpcoming() {
		t.Error("expected current maintenance to not be upcoming")
	}

	// Cancelled future maintenance - not upcoming (status is cancelled)
	m3, _ := NewMaintenance("Cancelled", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))
	m3.Cancel()
	if m3.IsUpcoming() {
		t.Error("expected cancelled maintenance to not be upcoming")
	}
}

func TestMaintenance_TimeUntilStart(t *testing.T) {
	now := time.Now()
	m, _ := NewMaintenance("Test", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))

	duration := m.TimeUntilStart()
	// Should be approximately 1 hour (allowing some tolerance)
	if duration < 59*time.Minute || duration > 61*time.Minute {
		t.Errorf("expected ~1 hour until start, got %v", duration)
	}
}

func TestMaintenance_TimeUntilEnd(t *testing.T) {
	now := time.Now()
	m, _ := NewMaintenance("Test", "Desc", now.Add(1*time.Hour), now.Add(2*time.Hour))

	duration := m.TimeUntilEnd()
	// Should be approximately 2 hours (allowing some tolerance)
	if duration < 119*time.Minute || duration > 121*time.Minute {
		t.Errorf("expected ~2 hours until end, got %v", duration)
	}
}
