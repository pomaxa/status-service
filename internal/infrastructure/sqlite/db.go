package sqlite

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps sql.DB with application-specific methods
type DB struct {
	*sql.DB
	path string
}

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// migrations is the list of all migrations in order
var migrations = []Migration{
	{
		Version: 1,
		Name:    "initial_schema",
		SQL: `
CREATE TABLE IF NOT EXISTS systems (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'green' CHECK(status IN ('green', 'yellow', 'red')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dependencies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    system_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'green' CHECK(status IN ('green', 'yellow', 'red')),
    heartbeat_url TEXT,
    heartbeat_interval INTEGER NOT NULL DEFAULT 0,
    last_check DATETIME,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_latency INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS status_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    system_id INTEGER,
    dependency_id INTEGER,
    old_status TEXT NOT NULL CHECK(old_status IN ('green', 'yellow', 'red')),
    new_status TEXT NOT NULL CHECK(new_status IN ('green', 'yellow', 'red')),
    message TEXT NOT NULL DEFAULT '',
    source TEXT NOT NULL DEFAULT 'manual' CHECK(source IN ('manual', 'heartbeat')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE SET NULL,
    FOREIGN KEY (dependency_id) REFERENCES dependencies(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_dependencies_system_id ON dependencies(system_id);
CREATE INDEX IF NOT EXISTS idx_status_log_system_id ON status_log(system_id);
CREATE INDEX IF NOT EXISTS idx_status_log_dependency_id ON status_log(dependency_id);
CREATE INDEX IF NOT EXISTS idx_status_log_created_at ON status_log(created_at);
CREATE INDEX IF NOT EXISTS idx_dependencies_heartbeat ON dependencies(heartbeat_url) WHERE heartbeat_url IS NOT NULL AND heartbeat_url != '';
`,
	},
	{
		Version: 2,
		Name:    "add_webhooks",
		SQL: `
CREATE TABLE IF NOT EXISTS webhooks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'generic' CHECK(type IN ('generic', 'slack', 'telegram', 'discord')),
    events TEXT NOT NULL DEFAULT '["status_change"]',
    system_ids TEXT,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhooks_enabled ON webhooks(enabled);
`,
	},
	{
		Version: 3,
		Name:    "add_maintenances",
		SQL: `
CREATE TABLE IF NOT EXISTS maintenances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    system_ids TEXT,
    status TEXT NOT NULL DEFAULT 'scheduled' CHECK(status IN ('scheduled', 'in_progress', 'completed', 'cancelled')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_maintenances_status ON maintenances(status);
CREATE INDEX IF NOT EXISTS idx_maintenances_start_time ON maintenances(start_time);
CREATE INDEX IF NOT EXISTS idx_maintenances_end_time ON maintenances(end_time);
`,
	},
	{
		Version: 4,
		Name:    "add_incidents",
		SQL: `
CREATE TABLE IF NOT EXISTS incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'investigating' CHECK(status IN ('investigating', 'identified', 'monitoring', 'resolved')),
    severity TEXT NOT NULL DEFAULT 'minor' CHECK(severity IN ('minor', 'major', 'critical')),
    system_ids TEXT,
    message TEXT NOT NULL DEFAULT '',
    postmortem TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME,
    acknowledged_at DATETIME,
    acknowledged_by TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS incident_updates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id INTEGER NOT NULL,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents(status);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents(created_at);
CREATE INDEX IF NOT EXISTS idx_incident_updates_incident_id ON incident_updates(incident_id);
`,
	},
	{
		Version: 5,
		Name:    "add_api_keys",
		SQL: `
CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    key_value TEXT NOT NULL UNIQUE,
    key_hash TEXT NOT NULL,
    scopes TEXT NOT NULL DEFAULT '["read"]',
    enabled BOOLEAN NOT NULL DEFAULT 1,
    expires_at DATETIME,
    last_used DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_value ON api_keys(key_value);
CREATE INDEX IF NOT EXISTS idx_api_keys_enabled ON api_keys(enabled);
`,
	},
	{
		Version: 6,
		Name:    "add_latency_history",
		SQL: `
CREATE TABLE IF NOT EXISTS latency_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    dependency_id INTEGER NOT NULL,
    latency_ms INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT 1,
    status_code INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (dependency_id) REFERENCES dependencies(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_latency_history_dependency_id ON latency_history(dependency_id);
CREATE INDEX IF NOT EXISTS idx_latency_history_created_at ON latency_history(created_at);
CREATE INDEX IF NOT EXISTS idx_latency_history_dep_time ON latency_history(dependency_id, created_at);
`,
	},
}

// New creates a new SQLite database connection
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, path: dbPath}, nil
}

// Migrate runs database migrations with version tracking and automatic backup
func (db *DB) Migrate() error {
	// Create schema_migrations table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Get current schema version
	currentVersion := db.getCurrentVersion()

	// Find pending migrations
	var pending []Migration
	for _, m := range migrations {
		if m.Version > currentVersion {
			pending = append(pending, m)
		}
	}

	if len(pending) == 0 {
		return nil // No migrations to apply
	}

	// Backup database before applying migrations
	if currentVersion > 0 {
		backupPath, err := db.backup()
		if err != nil {
			return fmt.Errorf("failed to backup database before migration: %w", err)
		}
		log.Printf("Database backed up to: %s", backupPath)
	}

	// Apply pending migrations
	for _, m := range pending {
		log.Printf("Applying migration %d: %s", m.Version, m.Name)

		if _, err := db.Exec(m.SQL); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.Version, m.Name, err)
		}

		// Record migration
		_, err := db.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			m.Version, m.Name,
		)
		if err != nil {
			return fmt.Errorf("failed to record migration %d: %w", m.Version, err)
		}
	}

	log.Printf("Applied %d migration(s), schema version: %d", len(pending), pending[len(pending)-1].Version)
	return nil
}

// getCurrentVersion returns the current schema version (0 if no migrations applied)
func (db *DB) getCurrentVersion() int {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		// Table might not exist yet for legacy databases
		return 0
	}
	return version
}

// backup creates a backup of the database file
func (db *DB) backup() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.backup-%s", db.path, timestamp)

	// Close any pending operations
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		log.Printf("Warning: WAL checkpoint failed: %v", err)
	}

	// Copy the database file
	if err := copyFile(db.path, backupPath); err != nil {
		return "", err
	}

	return backupPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination directory if needed
	if dir := filepath.Dir(dst); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}

// SchemaVersion returns the current schema version
func (db *DB) SchemaVersion() int {
	return db.getCurrentVersion()
}
