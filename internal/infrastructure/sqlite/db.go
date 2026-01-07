package sqlite

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps sql.DB with application-specific methods
type DB struct {
	*sql.DB
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

	return &DB{DB: db}, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	migrations := []string{
		createSystemsTable,
		createDependenciesTable,
		createStatusLogTable,
		createIndexes,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	// Add new columns if they don't exist (for existing databases)
	db.addColumnIfNotExists("systems", "url", "TEXT NOT NULL DEFAULT ''")
	db.addColumnIfNotExists("systems", "owner", "TEXT NOT NULL DEFAULT ''")

	return nil
}

// addColumnIfNotExists adds a column to a table if it doesn't exist
func (db *DB) addColumnIfNotExists(table, column, definition string) {
	// Check if column exists
	query := fmt.Sprintf("SELECT %s FROM %s LIMIT 1", column, table)
	_, err := db.Exec(query)
	if err != nil {
		// Column doesn't exist, add it
		alter := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
		db.Exec(alter)
	}
}

const createSystemsTable = `
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
`

const createDependenciesTable = `
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
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE
);
`

const createStatusLogTable = `
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
`

const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_dependencies_system_id ON dependencies(system_id);
CREATE INDEX IF NOT EXISTS idx_status_log_system_id ON status_log(system_id);
CREATE INDEX IF NOT EXISTS idx_status_log_dependency_id ON status_log(dependency_id);
CREATE INDEX IF NOT EXISTS idx_status_log_created_at ON status_log(created_at);
CREATE INDEX IF NOT EXISTS idx_dependencies_heartbeat ON dependencies(heartbeat_url) WHERE heartbeat_url IS NOT NULL AND heartbeat_url != '';
`
