package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNew verifies New opens a database file and pings successfully.
func TestNew(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if db.DB == nil {
		t.Fatal("expected non-nil underlying sql.DB")
	}
	if db.path != dbPath {
		t.Errorf("expected path %q, got %q", dbPath, db.path)
	}
}

// TestNew_InvalidPath forces a ping failure by pointing at a path whose parent
// directory does not exist, so the SQLite file cannot be created/opened.
func TestNew_InvalidPath(t *testing.T) {
	// A path inside a non-existent directory cannot be opened by SQLite.
	dbPath := filepath.Join(t.TempDir(), "nonexistent-dir", "test.db")

	db, err := New(dbPath)
	if err == nil {
		if db != nil {
			db.Close()
		}
		t.Fatal("expected error opening database in non-existent directory")
	}
}

// TestMigrate_FreshDatabase applies all migrations to an empty file-backed DB
// and verifies the schema version advances to the latest migration.
func TestMigrate_FreshDatabase(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	want := migrations[len(migrations)-1].Version
	if got := db.SchemaVersion(); got != want {
		t.Errorf("SchemaVersion() = %d, want %d", got, want)
	}
}

// TestMigrate_Idempotent verifies a second Migrate() on an already-migrated DB
// is a no-op (no pending migrations) and also exercises the backup branch that
// runs when currentVersion > 0 and new migrations would be applied.
func TestMigrate_Idempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "idempotent.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("first Migrate() error = %v", err)
	}
	v1 := db.SchemaVersion()

	// Second run should be a no-op with no pending migrations.
	if err := db.Migrate(); err != nil {
		t.Fatalf("second Migrate() error = %v", err)
	}
	v2 := db.SchemaVersion()

	if v1 != v2 {
		t.Errorf("schema version changed on idempotent migrate: %d -> %d", v1, v2)
	}
}

// TestMigrate_WithBackup exercises the backup branch: we apply only the first
// migration manually (recording it), then run Migrate() which sees
// currentVersion > 0 and pending migrations, triggering db.backup().
func TestMigrate_WithBackup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "backup.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Create the migrations table and record version 1 as already applied.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("failed to create schema_migrations: %v", err)
	}
	if _, err := db.Exec(migrations[0].SQL); err != nil {
		t.Fatalf("failed to apply first migration SQL: %v", err)
	}
	if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'initial_schema')"); err != nil {
		t.Fatalf("failed to record migration 1: %v", err)
	}

	if db.getCurrentVersion() != 1 {
		t.Fatalf("expected current version 1 before migrate, got %d", db.getCurrentVersion())
	}

	// Migrate now sees currentVersion=1 and pending migrations 2..N, which
	// triggers the backup path.
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	want := migrations[len(migrations)-1].Version
	if got := db.SchemaVersion(); got != want {
		t.Errorf("SchemaVersion() = %d, want %d", got, want)
	}

	// A backup file should have been produced next to the DB.
	matches, _ := filepath.Glob(dbPath + ".backup-*")
	if len(matches) == 0 {
		t.Error("expected a backup file to be created during migration")
	}
}

// TestGetCurrentVersion_NoTable verifies getCurrentVersion returns 0 when the
// schema_migrations table does not exist (legacy database path).
func TestGetCurrentVersion_NoTable(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "noversion.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if v := db.getCurrentVersion(); v != 0 {
		t.Errorf("getCurrentVersion() = %d, want 0 when table missing", v)
	}
	if v := db.SchemaVersion(); v != 0 {
		t.Errorf("SchemaVersion() = %d, want 0 when table missing", v)
	}
}

// TestMigrate_CreateTableError forces the schema_migrations CREATE TABLE to fail
// by closing the connection before calling Migrate.
func TestMigrate_CreateTableError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "migrate-err.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	db.Close()

	if err := db.Migrate(); err == nil {
		t.Fatal("expected error from Migrate() on closed database")
	}
}

// TestBackup verifies backup() copies the database file and returns its path.
func TestBackup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "tobackup.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}

	backupPath, err := db.backup()
	if err != nil {
		t.Fatalf("backup() error = %v", err)
	}

	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("expected backup file at %q: %v", backupPath, err)
	}
}

// TestBackup_CopyError forces backup()'s copyFile to fail because the source
// database file does not exist on disk (in-memory style path).
func TestBackup_CopyError(t *testing.T) {
	// Point path at a file that will never exist; the WAL checkpoint warning is
	// logged but copyFile must then fail opening the source.
	missing := filepath.Join(t.TempDir(), "never-created.db")

	db, err := New(missing)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Remove the file that New() created so copyFile's os.Open fails.
	_ = os.Remove(missing)
	_ = os.Remove(missing + "-wal")
	_ = os.Remove(missing + "-shm")

	if _, err := db.backup(); err == nil {
		t.Fatal("expected backup() to fail when source file is missing")
	}
}

// TestCopyFile verifies copyFile copies bytes from src to dst, creating the
// destination directory when needed.
func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	content := []byte("hello copy file")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}

	// Destination inside a subdirectory that does not exist yet.
	dst := filepath.Join(dir, "subdir", "dst.txt")

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read dst: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

// TestCopyFile_SourceMissing covers the os.Open error branch.
func TestCopyFile_SourceMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "does-not-exist.txt")
	dst := filepath.Join(dir, "out.txt")

	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected error copying a missing source file")
	}
}

// TestCopyFile_MkdirError covers the MkdirAll error branch: the destination's
// parent path component is an existing regular file, so MkdirAll fails.
func TestCopyFile_MkdirError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}

	// Create a regular file, then try to use it as a directory in dst's path.
	fileAsDir := filepath.Join(dir, "afile")
	if err := os.WriteFile(fileAsDir, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	dst := filepath.Join(fileAsDir, "child", "dst.txt")

	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected error creating destination under a non-directory path")
	}
}

// TestCopyFile_CreateError covers the os.Create error branch: the destination
// path is itself an existing directory, so Create fails after MkdirAll succeeds.
func TestCopyFile_CreateError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write src: %v", err)
	}

	// dst is an existing directory: filepath.Dir(dst) exists (MkdirAll no-op),
	// but os.Create(dst) fails because dst is a directory.
	dst := filepath.Join(dir, "iam-a-dir")
	if err := os.Mkdir(dst, 0755); err != nil {
		t.Fatalf("failed to mkdir dst: %v", err)
	}

	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected error creating destination that is a directory")
	}
}

// TestBackup_WALCheckpointWarning closes the DB before calling backup. The WAL
// checkpoint PRAGMA then fails (logged warning, non-fatal), but the underlying
// file still exists so the file copy succeeds.
func TestBackup_WALCheckpointWarning(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "wal.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := db.Migrate(); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	// Closing makes the PRAGMA wal_checkpoint Exec fail (warning branch), while
	// the on-disk file remains for copyFile to read.
	db.Close()

	backupPath, err := db.backup()
	if err != nil {
		t.Fatalf("backup() error = %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("expected backup file at %q: %v", backupPath, err)
	}
}

// TestMigrate_BackupFailure exercises the branch where Migrate must back up the
// database (currentVersion > 0 with pending migrations) but the backup fails
// because the on-disk source file is missing.
func TestMigrate_BackupFailure(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "bkpfail.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Record version 1 as applied so Migrate sees pending migrations 2..N.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("failed to create schema_migrations: %v", err)
	}
	if _, err := db.Exec(migrations[0].SQL); err != nil {
		t.Fatalf("failed to apply first migration: %v", err)
	}
	if _, err := db.Exec("INSERT INTO schema_migrations (version, name) VALUES (1, 'initial_schema')"); err != nil {
		t.Fatalf("failed to record migration 1: %v", err)
	}

	// Point the path at a missing file so backup()'s copyFile fails, while the
	// live (in-memory page cache backed) connection still answers queries.
	db.path = filepath.Join(t.TempDir(), "this-file-does-not-exist.db")

	if err := db.Migrate(); err == nil {
		t.Fatal("expected Migrate() to fail when backup cannot read source file")
	}
}

// TestMigrate_SQLAndRecordErrors injects a deliberately-broken migration with a
// version above the latest so Migrate attempts to apply it and the Exec fails.
// It restores the package-level migrations slice afterward.
func TestMigrate_MigrationSQLError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "sqlerr.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	// Bring the DB up to the current latest version first.
	if err := db.Migrate(); err != nil {
		t.Fatalf("initial Migrate() error = %v", err)
	}

	orig := migrations
	defer func() { migrations = orig }()

	bad := Migration{
		Version: orig[len(orig)-1].Version + 1,
		Name:    "broken_migration",
		SQL:     "THIS IS NOT VALID SQL;",
	}
	migrations = append(append([]Migration{}, orig...), bad)

	if err := db.Migrate(); err == nil {
		t.Fatal("expected Migrate() to fail applying broken migration SQL")
	}
}

// TestMigrate_RecordError injects a migration whose SQL succeeds (it drops the
// schema_migrations table) so the subsequent "INSERT INTO schema_migrations"
// recording statement fails, hitting the "failed to record migration" branch.
func TestMigrate_RecordError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "recerr.db")

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("initial Migrate() error = %v", err)
	}

	orig := migrations
	defer func() { migrations = orig }()

	latest := orig[len(orig)-1].Version
	// The migration SQL drops schema_migrations: the Exec succeeds, but Migrate
	// then tries to INSERT the migration record into a table that no longer
	// exists, which fails.
	sneaky := Migration{
		Version: latest + 1,
		Name:    "drops_tracking_table",
		SQL:     "DROP TABLE schema_migrations;",
	}
	migrations = append(append([]Migration{}, orig...), sneaky)

	if err := db.Migrate(); err == nil {
		t.Fatal("expected Migrate() to fail recording migration after table dropped")
	}
}
