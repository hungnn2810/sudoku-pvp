package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// PostgreSQL database driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// File-system source driver for golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending SQL migrations from db/migrations/ to the
// database at dsn. It is safe to call multiple times: migrate.ErrNoChange is
// treated as success (idempotent). Wrap with context.WithTimeout(30s) at
// startup per RESEARCH.md Pitfall 3.
func RunMigrations(dsn string) error {
	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		return fmt.Errorf("migrate new: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}
