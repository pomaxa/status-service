package domain

import "errors"

// Sentinel error categories used to map domain/application failures to the
// correct HTTP status at the interface boundary:
//
//   - ErrNotFound   -> 404
//   - ErrValidation -> 400 (the wrapped message is user-actionable)
//   - anything else -> 500 (treated as internal; detail logged, not returned)
//
// errors.Is traverses %w chains, so a service may wrap these and the boundary
// classification still holds.
var (
	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrValidation indicates user-supplied input failed validation.
	ErrValidation = errors.New("validation failed")
)

// validationError is a validation failure that reports its own human-readable
// message while still matching ErrValidation via errors.Is. This lets existing
// error text (and any test asserting it) stay unchanged while enabling
// category-based HTTP status mapping.
type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func (e *validationError) Is(target error) bool { return target == ErrValidation }

// ValidationError builds a validation error with the given message. It satisfies
// errors.Is(err, ErrValidation).
func ValidationError(msg string) error { return &validationError{msg: msg} }
