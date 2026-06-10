package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestValidationError_IsAndMessage(t *testing.T) {
	e := ValidationError("title is required")
	if e.Error() != "title is required" {
		t.Errorf("Error() = %q, want %q", e.Error(), "title is required")
	}
	if !errors.Is(e, ErrValidation) {
		t.Error("ValidationError should match ErrValidation via errors.Is")
	}
	if errors.Is(e, ErrNotFound) {
		t.Error("ValidationError must not match ErrNotFound")
	}
	// errors.Is must still see through a %w wrap (as services do).
	wrapped := fmt.Errorf("invalid system data: %w", e)
	if !errors.Is(wrapped, ErrValidation) {
		t.Error("wrapped ValidationError should still match ErrValidation")
	}
}

// TestDomainValidationSentinelsAreTagged ensures the converted domain validation
// sentinels classify as ErrValidation (so the HTTP layer maps them to 400).
func TestDomainValidationSentinelsAreTagged(t *testing.T) {
	for _, e := range []error{
		ErrInvalidStatus,
		ErrEmptyName,
		ErrInvalidSystemID,
		ErrInvalidHeartbeatURL,
		ErrInvalidHeartbeatInterval,
		ErrInvalidHeartbeatMethod,
		ErrInvalidExpectStatus,
		ErrInvalidExpectBody,
		ErrBlockedHeartbeatURL,
	} {
		if !errors.Is(e, ErrValidation) {
			t.Errorf("%q should match ErrValidation", e.Error())
		}
	}
}
