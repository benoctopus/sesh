package db

import (
	"database/sql"
	_ "embed"

	"github.com/rotisserie/eris"
)

// Embed migration files
//
//go:embed migrations/001_initial_schema.sql
var migration001 string

//go:embed migrations/002_session_history.sql
var migration002 string

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB) error {
	// Create schema_migrations table if it doesn't exist
	// Note: We need to check if it exists first since the migration itself creates it
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return eris.Wrap(err, "failed to create schema_migrations table")
	}

	// Define all migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{version: 1, sql: migration001},
		{version: 2, sql: migration002},
	}

	// Apply each migration if not already applied
	for _, m := range migrations {
		applied, err := isMigrationApplied(db, m.version)
		if err != nil {
			return eris.Wrapf(err, "failed to check migration %d", m.version)
		}

		if applied {
			continue
		}

		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return eris.Wrapf(err, "failed to begin transaction for migration %d", m.version)
		}

		// Execute the migration SQL
		if _, err := tx.Exec(m.sql); err != nil {
			//nolint:errcheck // Rollback in error path
			tx.Rollback()
			return eris.Wrapf(err, "failed to execute migration %d", m.version)
		}

		// Record migration as applied
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			//nolint:errcheck // Rollback in error path
			tx.Rollback()
			return eris.Wrapf(err, "failed to record migration %d", m.version)
		}

		if err := tx.Commit(); err != nil {
			return eris.Wrapf(err, "failed to commit migration %d", m.version)
		}
	}

	return nil
}

// isMigrationApplied checks if a migration version has been applied
func isMigrationApplied(db *sql.DB, version int) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
	if err != nil {
		return false, eris.Wrap(err, "failed to query schema_migrations")
	}
	return count > 0, nil
}
