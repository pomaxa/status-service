package sqlite

import (
	"context"
	"testing"
	"time"

	"status-incident/internal/domain"
)

func TestAPIKeyRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_test_1234567890abcdef"
	key := &domain.APIKey{
		Name:    "Test API Key",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read", "write"},
		Enabled: true,
	}

	err := repo.Create(ctx, key)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if key.ID == 0 {
		t.Error("expected key ID to be set after Create()")
	}
}

func TestAPIKeyRepo_Create_WithExpiry(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	expiry := time.Now().Add(30 * 24 * time.Hour) // 30 days
	keyValue := "sk_expiring_1234567890"
	key := &domain.APIKey{
		Name:      "Expiring Key",
		Key:       keyValue,
		KeyHash:   domain.HashAPIKey(keyValue),
		Scopes:    []string{"read"},
		Enabled:   true,
		ExpiresAt: &expiry,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByKey() returned nil")
	}

	if retrieved.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestAPIKeyRepo_GetByKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_unique_key_123456"
	keyHash := domain.HashAPIKey(keyValue)
	key := &domain.APIKey{
		Name:    "Get By Key Test",
		Key:     keyValue,
		KeyHash: keyHash,
		Scopes:  []string{"read", "write", "admin"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, keyValue)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved == nil {
		t.Fatal("GetByKey() returned nil")
	}

	if retrieved.ID != key.ID {
		t.Errorf("ID = %d, want %d", retrieved.ID, key.ID)
	}
	if retrieved.Name != key.Name {
		t.Errorf("Name = %s, want %s", retrieved.Name, key.Name)
	}
	if retrieved.KeyHash != keyHash {
		t.Errorf("KeyHash = %s, want %s", retrieved.KeyHash, keyHash)
	}
	if len(retrieved.Scopes) != 3 {
		t.Errorf("Scopes count = %d, want 3", len(retrieved.Scopes))
	}
	if !retrieved.Enabled {
		t.Error("expected Enabled = true")
	}
}

func TestAPIKeyRepo_GetByKey_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	retrieved, err := repo.GetByKey(ctx, "sk_nonexistent_key")
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved != nil {
		t.Error("expected nil for non-existent key")
	}
}

func TestAPIKeyRepo_GetByKey_LegacyRandomHashBackfill(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_legacy_key_backfill"
	legacyHash := "legacy-random-hash-that-does-not-match"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO api_keys (name, key_value, key_hash, scopes, enabled)
		VALUES (?, ?, ?, ?, ?)
	`, "Legacy Key", keyValue, legacyHash, `["read"]`, true); err != nil {
		t.Fatalf("failed to seed legacy key: %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, keyValue)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if retrieved == nil {
		t.Fatal("GetByKey() returned nil for legacy key")
	}

	expectedHash := domain.HashAPIKey(keyValue)
	if retrieved.KeyHash != expectedHash {
		t.Errorf("retrieved KeyHash = %s, want %s", retrieved.KeyHash, expectedHash)
	}

	var storedHash string
	if err := db.QueryRowContext(ctx, `
		SELECT key_hash
		FROM api_keys
		WHERE key_value = ?
	`, keyValue).Scan(&storedHash); err != nil {
		t.Fatalf("failed to read stored hash: %v", err)
	}
	if storedHash != expectedHash {
		t.Errorf("stored key_hash = %s, want %s", storedHash, expectedHash)
	}
}

func TestAPIKeyRepo_GetAll(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	// Create multiple API keys
	keys := []struct {
		name   string
		key    string
		scopes []string
	}{
		{"Key 1", "sk_key_1", []string{"read"}},
		{"Key 2", "sk_key_2", []string{"read", "write"}},
		{"Key 3", "sk_key_3", []string{"admin"}},
	}

	for _, k := range keys {
		key := &domain.APIKey{
			Name:    k.name,
			Key:     k.key,
			KeyHash: domain.HashAPIKey(k.key),
			Scopes:  k.scopes,
			Enabled: true,
		}
		if err := repo.Create(ctx, key); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		time.Sleep(time.Millisecond)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}

	if len(all) != 3 {
		t.Errorf("GetAll() returned %d keys, want 3", len(all))
	}

	// Should be ordered by created_at DESC (most recent first)
	if all[0].Name != "Key 3" {
		t.Errorf("first key name = %s, want Key 3", all[0].Name)
	}
}

func TestAPIKeyRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_update_test"
	key := &domain.APIKey{
		Name:    "Original Name",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	key.Name = "Updated Name"
	key.Scopes = []string{"read", "write", "admin"}
	key.Enabled = false
	expiry := time.Now().Add(7 * 24 * time.Hour)
	key.ExpiresAt = &expiry

	if err := repo.Update(ctx, key); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	retrieved, err := repo.GetByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %s, want Updated Name", retrieved.Name)
	}
	if len(retrieved.Scopes) != 3 {
		t.Errorf("Scopes count = %d, want 3", len(retrieved.Scopes))
	}
	if retrieved.Enabled {
		t.Error("expected Enabled = false")
	}
	if retrieved.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestAPIKeyRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_delete_test"
	key := &domain.APIKey{
		Name:    "To Delete",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := repo.Delete(ctx, key.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}
	if retrieved != nil {
		t.Error("expected nil after deletion")
	}
}

func TestAPIKeyRepo_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_lastused_test"
	key := &domain.APIKey{
		Name:    "Last Used Test",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Initially, LastUsed should be nil
	retrieved, _ := repo.GetByKey(ctx, key.Key)
	if retrieved.LastUsed != nil {
		t.Error("expected LastUsed to be nil initially")
	}

	// Update last used
	if err := repo.UpdateLastUsed(ctx, key.ID); err != nil {
		t.Fatalf("UpdateLastUsed() error = %v", err)
	}

	// Verify
	retrieved, _ = repo.GetByKey(ctx, key.Key)
	if retrieved.LastUsed == nil {
		t.Error("expected LastUsed to be set after UpdateLastUsed()")
	}
}

func TestAPIKeyRepo_ScopesSerialization(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_scopes_test"
	scopes := []string{"read", "write", "admin", "delete", "custom_scope"}
	key := &domain.APIKey{
		Name:    "Scopes Test",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  scopes,
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if len(retrieved.Scopes) != len(scopes) {
		t.Errorf("Scopes count = %d, want %d", len(retrieved.Scopes), len(scopes))
	}

	scopeMap := make(map[string]bool)
	for _, s := range retrieved.Scopes {
		scopeMap[s] = true
	}

	for _, expected := range scopes {
		if !scopeMap[expected] {
			t.Errorf("missing scope: %s", expected)
		}
	}
}

func TestAPIKeyRepo_EnabledDisabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	// Create enabled key
	enabledKeyValue := "sk_enabled"
	enabledKey := &domain.APIKey{
		Name:    "Enabled Key",
		Key:     enabledKeyValue,
		KeyHash: domain.HashAPIKey(enabledKeyValue),
		Scopes:  []string{"read"},
		Enabled: true,
	}
	repo.Create(ctx, enabledKey)

	// Create disabled key
	disabledKeyValue := "sk_disabled"
	disabledKey := &domain.APIKey{
		Name:    "Disabled Key",
		Key:     disabledKeyValue,
		KeyHash: domain.HashAPIKey(disabledKeyValue),
		Scopes:  []string{"read"},
		Enabled: false,
	}
	repo.Create(ctx, disabledKey)

	// Verify enabled
	enabled, _ := repo.GetByKey(ctx, "sk_enabled")
	if !enabled.Enabled {
		t.Error("expected enabled key to be enabled")
	}

	// Verify disabled
	disabled, _ := repo.GetByKey(ctx, "sk_disabled")
	if disabled.Enabled {
		t.Error("expected disabled key to be disabled")
	}
}

func TestAPIKeyRepo_NullableFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_minimal"
	// Create key without optional fields
	key := &domain.APIKey{
		Name:    "Minimal Key",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read"},
		Enabled: true,
		// ExpiresAt is nil
		// LastUsed is nil
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := repo.GetByKey(ctx, key.Key)
	if err != nil {
		t.Fatalf("GetByKey() error = %v", err)
	}

	if retrieved.ExpiresAt != nil {
		t.Error("expected ExpiresAt to be nil")
	}
	if retrieved.LastUsed != nil {
		t.Error("expected LastUsed to be nil")
	}
}

func TestAPIKeyRepo_HasScope(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_scope_check"
	key := &domain.APIKey{
		Name:    "Scope Check",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read", "write"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, _ := repo.GetByKey(ctx, key.Key)

	if !retrieved.HasScope("read") {
		t.Error("expected HasScope('read') = true")
	}
	if !retrieved.HasScope("write") {
		t.Error("expected HasScope('write') = true")
	}
	if retrieved.HasScope("admin") {
		t.Error("expected HasScope('admin') = false")
	}
}

func TestAPIKeyRepo_HasScope_Admin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_admin"
	// Admin scope should grant access to all
	key := &domain.APIKey{
		Name:    "Admin Key",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"admin"},
		Enabled: true,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, _ := repo.GetByKey(ctx, key.Key)

	// Admin scope should include all other scopes
	if !retrieved.HasScope("read") {
		t.Error("expected admin to have read scope")
	}
	if !retrieved.HasScope("write") {
		t.Error("expected admin to have write scope")
	}
	if !retrieved.HasScope("admin") {
		t.Error("expected admin to have admin scope")
	}
}

func TestAPIKeyRepo_IsExpired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	// Create expired key
	expiredKeyValue := "sk_expired"
	pastExpiry := time.Now().Add(-24 * time.Hour)
	expiredKey := &domain.APIKey{
		Name:      "Expired Key",
		Key:       expiredKeyValue,
		KeyHash:   domain.HashAPIKey(expiredKeyValue),
		Scopes:    []string{"read"},
		Enabled:   true,
		ExpiresAt: &pastExpiry,
	}
	repo.Create(ctx, expiredKey)

	// Create non-expired key
	validKeyValue := "sk_valid"
	futureExpiry := time.Now().Add(24 * time.Hour)
	validKey := &domain.APIKey{
		Name:      "Valid Key",
		Key:       validKeyValue,
		KeyHash:   domain.HashAPIKey(validKeyValue),
		Scopes:    []string{"read"},
		Enabled:   true,
		ExpiresAt: &futureExpiry,
	}
	repo.Create(ctx, validKey)

	// Create key without expiry
	noExpiryKeyValue := "sk_no_expiry"
	noExpiryKey := &domain.APIKey{
		Name:    "No Expiry Key",
		Key:     noExpiryKeyValue,
		KeyHash: domain.HashAPIKey(noExpiryKeyValue),
		Scopes:  []string{"read"},
		Enabled: true,
	}
	repo.Create(ctx, noExpiryKey)

	// Check expired
	expired, _ := repo.GetByKey(ctx, "sk_expired")
	if !expired.IsExpired() {
		t.Error("expected expired key to be expired")
	}
	if expired.IsValid() {
		t.Error("expected expired key to be invalid")
	}

	// Check valid
	valid, _ := repo.GetByKey(ctx, "sk_valid")
	if valid.IsExpired() {
		t.Error("expected valid key to not be expired")
	}
	if !valid.IsValid() {
		t.Error("expected valid key to be valid")
	}

	// Check no expiry
	noExpiry, _ := repo.GetByKey(ctx, "sk_no_expiry")
	if noExpiry.IsExpired() {
		t.Error("expected key without expiry to not be expired")
	}
	if !noExpiry.IsValid() {
		t.Error("expected key without expiry to be valid")
	}
}

func TestAPIKeyRepo_IsValid_DisabledKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewAPIKeyRepo(db)
	ctx := context.Background()

	keyValue := "sk_disabled_check"
	key := &domain.APIKey{
		Name:    "Disabled Key",
		Key:     keyValue,
		KeyHash: domain.HashAPIKey(keyValue),
		Scopes:  []string{"read"},
		Enabled: false,
	}

	if err := repo.Create(ctx, key); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, _ := repo.GetByKey(ctx, key.Key)

	if retrieved.IsValid() {
		t.Error("expected disabled key to be invalid")
	}
}
