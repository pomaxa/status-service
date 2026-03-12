package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled      bool
	Username     string
	PasswordHash string // bcrypt hash or plain for simplicity
	APIKeys      []APIKey
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        int64
	Name      string     // descriptive name like "CI/CD Pipeline"
	Key       string     // the actual key (stored hashed)
	KeyHash   string     // hash of the key for comparison
	Scopes    []string   // allowed scopes: "read", "write", "admin"
	CreatedAt time.Time
	LastUsed  *time.Time
	ExpiresAt *time.Time
	Enabled   bool
}

// HasScope checks if API key has required scope
func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == "admin" {
			return true
		}
	}
	return false
}

// IsExpired checks if API key has expired
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid checks if API key is usable
func (k *APIKey) IsValid() bool {
	return k.Enabled && !k.IsExpired()
}

// GenerateAPIKey creates a new random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_" + hex.EncodeToString(bytes), nil
}

// HashAPIKey creates a SHA-256 hash of the API key for secure storage
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// CompareAPIKey securely compares API key with stored hash
func CompareAPIKey(provided, storedHash string) bool {
	providedHash := HashAPIKey(provided)
	return subtle.ConstantTimeCompare([]byte(providedHash), []byte(storedHash)) == 1
}

// User represents an authenticated user context
type User struct {
	Username string
	IsAPIKey bool
	APIKeyID int64
	Scopes   []string
}

// UserContextKey is the context key for user
type userContextKey struct{}

// UserFromContext extracts user from context
func UserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value(userContextKey{}).(*User); ok {
		return user
	}
	return nil
}

// ContextWithUser adds user to context
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// APIKeyRepository defines operations for API key persistence
type APIKeyRepository interface {
	// Create persists a new API key
	Create(ctx context.Context, key *APIKey) error

	// GetByKey retrieves API key by the key value
	GetByKey(ctx context.Context, key string) (*APIKey, error)

	// GetAll retrieves all API keys
	GetAll(ctx context.Context) ([]*APIKey, error)

	// Update saves changes to an API key
	Update(ctx context.Context, key *APIKey) error

	// Delete removes an API key
	Delete(ctx context.Context, id int64) error

	// UpdateLastUsed updates the last used timestamp
	UpdateLastUsed(ctx context.Context, id int64) error
}
