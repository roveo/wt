package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var defaultDB *sql.DB

// Open opens the database at the default location (~/.local/share/wt/wt.db)
func Open() (*sql.DB, error) {
	dbPath, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return OpenAt(dbPath)
}

// OpenAt opens the database at the specified path
func OpenAt(path string) (*sql.DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// DefaultPath returns the default database path
func DefaultPath() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "wt", "wt.db"), nil
}

// Default returns the default database connection, opening it if necessary
func Default() (*sql.DB, error) {
	if defaultDB != nil {
		return defaultDB, nil
	}
	var err error
	defaultDB, err = Open()
	return defaultDB, err
}

// Close closes the default database connection
func Close() error {
	if defaultDB != nil {
		err := defaultDB.Close()
		defaultDB = nil
		return err
	}
	return nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS repos (
		id INTEGER PRIMARY KEY,
		path TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		worktrees_dir TEXT NOT NULL,
		last_synced_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS worktrees (
		id INTEGER PRIMARY KEY,
		repo_id INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
		path TEXT UNIQUE NOT NULL,
		branch TEXT NOT NULL,
		is_main BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME
	);

	CREATE INDEX IF NOT EXISTS idx_worktrees_repo_id ON worktrees(repo_id);
	CREATE INDEX IF NOT EXISTS idx_worktrees_deleted_at ON worktrees(deleted_at);
	CREATE INDEX IF NOT EXISTS idx_repos_deleted_at ON repos(deleted_at);
	`
	_, err := db.Exec(schema)
	return err
}
