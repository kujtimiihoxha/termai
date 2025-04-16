package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/mattn/go-sqlite3"

	"github.com/kujtimiihoxha/opencode/internal/config"
	"github.com/kujtimiihoxha/opencode/internal/logging"
)

func Connect() (*sql.DB, error) {
	dataDir := config.Get().Data.Directory
	if dataDir == "" {
		return nil, fmt.Errorf("data.dir is not set")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	dbPath := filepath.Join(dataDir, "opencode.db")
	// Open the SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection
	if err = db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Set pragmas for better performance
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA page_size = 4096;",
		"PRAGMA cache_size = -8000;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, pragma := range pragmas {
		if _, err = db.Exec(pragma); err != nil {
			logging.Warn("Failed to set pragma", pragma, err)
		} else {
			logging.Warn("Set pragma", "pragma", pragma)
		}
	}

	// Initialize schema from embedded file
	d, err := iofs.New(FS, "migrations")
	if err != nil {
		logging.Error("Failed to open embedded migrations", "error", err)
		db.Close()
		return nil, fmt.Errorf("failed to open embedded migrations: %w", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		logging.Error("Failed to create SQLite driver", "error", err)
		db.Close()
		return nil, fmt.Errorf("failed to create SQLite driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "ql", driver)
	if err != nil {
		logging.Error("Failed to create migration instance", "error", err)
		db.Close()
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		logging.Error("Migration failed", "error", err)
		db.Close()
		return nil, fmt.Errorf("failed to apply schema: %w", err)
	} else if err == migrate.ErrNoChange {
		logging.Info("No schema changes to apply")
	} else {
		logging.Info("Schema migration applied successfully")
	}

	return db, nil
}
