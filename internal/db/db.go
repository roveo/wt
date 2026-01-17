package db

import (
	"database/sql"
	"embed"
	"errors"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

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

	if err := runMigrations(db); err != nil {
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

func runMigrations(db *sql.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}

	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite3", dbDriver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	return nil
}
