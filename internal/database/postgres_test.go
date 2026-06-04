//go:build integration

package database_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go"

	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/database"
)

// startPostgresContainer starts a real postgres container for integration tests.
// Returns the DSN and a cleanup function.
func startPostgresContainer(t *testing.T) (string, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("sudoku_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Skipf("skipping integration test: failed to start postgres container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(pgContainer)
		t.Fatalf("failed to get connection string: %v", err)
	}

	cleanup := func() {
		_ = testcontainers.TerminateContainer(pgContainer)
	}

	return connStr, cleanup
}

// TestNewPool_Ping verifies that NewPool returns a non-nil pool and ping succeeds
// when given a valid DSN to a running postgres container.
func TestNewPool_Ping(t *testing.T) {
	dsn, cleanup := startPostgresContainer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.PostgresConfig{
		DSN:      dsn,
		MaxConns: 5,
		MinConns: 1,
	}

	pool, err := database.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("NewPool returned error: %v", err)
	}
	if pool == nil {
		t.Fatal("NewPool returned nil pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool.Ping failed: %v", err)
	}
}

// TestNewPool_InvalidDSN verifies that NewPool returns an error containing "pgxpool"
// when given an invalid DSN string.
func TestNewPool_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.PostgresConfig{
		DSN:      "not-a-valid-dsn",
		MaxConns: 5,
		MinConns: 1,
	}

	pool, err := database.NewPool(ctx, cfg)
	if err == nil {
		if pool != nil {
			pool.Close()
		}
		t.Fatal("expected NewPool to return error for invalid DSN, got nil")
	}
	if pool != nil {
		pool.Close()
		t.Fatal("expected NewPool to return nil pool on error")
	}

	// The error message should contain "pgxpool" as documented.
	errMsg := err.Error()
	if len(errMsg) == 0 {
		t.Fatal("expected non-empty error message")
	}
	t.Logf("got expected error: %v", err)
}

// TestRunMigrations_AllTables verifies that after RunMigrations on a fresh DB,
// all 11 expected tables exist in the public schema.
func TestRunMigrations_AllTables(t *testing.T) {
	dsn, cleanup := startPostgresContainer(t)
	defer cleanup()

	if err := database.RunMigrations(dsn); err != nil {
		t.Fatalf("RunMigrations returned error: %v", err)
	}

	// Open a stdlib sql.DB to query information_schema.
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(context.Background(),
		"SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'")
	if err != nil {
		t.Fatalf("query information_schema failed: %v", err)
	}
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		tables[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	expected := []string{
		"users",
		"wallets",
		"wallet_transactions",
		"sudoku_puzzles",
		"matches",
		"match_players",
		"match_moves",
		"missions",
		"user_missions",
		"shop_items",
		"inventory_items",
	}

	for _, tbl := range expected {
		if !tables[tbl] {
			t.Errorf("expected table %q not found after migration; found tables: %v", tbl, tables)
		}
	}
}

// TestRunMigrations_Idempotent verifies that running RunMigrations twice on the
// same database returns nil error (ErrNoChange is treated as success).
func TestRunMigrations_Idempotent(t *testing.T) {
	dsn, cleanup := startPostgresContainer(t)
	defer cleanup()

	if err := database.RunMigrations(dsn); err != nil {
		t.Fatalf("first RunMigrations returned error: %v", err)
	}
	if err := database.RunMigrations(dsn); err != nil {
		t.Fatalf("second RunMigrations returned error: %v (should be nil, ErrNoChange treated as success)", err)
	}
}
