//go:build integration

package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/database"
	"sudoku-pvp/internal/testutil"
)

// startRawPostgresContainer starts a plain Postgres container without running migrations.
// Returns the DSN and a cleanup function. Useful for tests that exercise RunMigrations directly.
func startRawPostgresContainer(t *testing.T) (string, func()) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("sudoku_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	cleanup := func() {
		require.NoError(t, testcontainers.TerminateContainer(ctr), "failed to terminate container")
	}

	return connStr, cleanup
}

// TestNewPool_Ping verifies that NewPool returns a non-nil pool and ping succeeds
// when given a valid DSN to a running postgres container.
func TestNewPool_Ping(t *testing.T) {
	dsn, cleanup := startRawPostgresContainer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := config.PostgresConfig{
		DSN:      dsn,
		MaxConns: 5,
		MinConns: 1,
	}

	pool, err := database.NewPool(ctx, cfg)
	require.NoError(t, err, "NewPool returned error")
	require.NotNil(t, pool, "NewPool returned nil pool")
	defer pool.Close()

	require.NoError(t, pool.Ping(ctx), "pool.Ping failed")
}

// TestNewPool_InvalidDSN verifies that NewPool returns an error for an invalid DSN.
func TestNewPool_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.PostgresConfig{
		DSN:      "not-a-valid-dsn",
		MaxConns: 5,
		MinConns: 1,
	}

	pool, err := database.NewPool(ctx, cfg)
	if pool != nil {
		pool.Close()
	}
	require.Error(t, err, "expected NewPool to return error for invalid DSN")
	require.Nil(t, pool, "expected nil pool on error")
}

// TestMigrations verifies that after RunMigrations on a fresh DB, all 11 expected
// tables exist in the public schema. Uses SetupPostgres which runs migrations internally.
func TestMigrations(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupPostgres(ctx, t)
	defer cleanup()

	rows, err := pool.Query(ctx,
		"SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE' ORDER BY table_name")
	require.NoError(t, err, "querying information_schema.tables failed")
	defer rows.Close()

	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables[name] = true
	}
	require.NoError(t, rows.Err())

	expected := []string{
		"inventory_items",
		"match_moves",
		"match_players",
		"matches",
		"missions",
		"shop_items",
		"sudoku_puzzles",
		"user_missions",
		"users",
		"wallet_transactions",
		"wallets",
	}

	for _, tbl := range expected {
		require.True(t, tables[tbl], "expected table %q not found after migration; found tables: %v", tbl, tables)
	}
}

// TestSchemaComplete verifies the users table schema matches DATABASE_SCHEMA.md.
// Checks that all required columns exist and that the id column is of type uuid.
func TestSchemaComplete(t *testing.T) {
	ctx := context.Background()
	pool, cleanup := testutil.SetupPostgres(ctx, t)
	defer cleanup()

	type columnInfo struct {
		name     string
		dataType string
	}

	rows, err := pool.Query(ctx,
		`SELECT column_name, data_type
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = 'users'
		 ORDER BY column_name`)
	require.NoError(t, err, "querying information_schema.columns failed")
	defer rows.Close()

	cols := make(map[string]string)
	for rows.Next() {
		var name, dataType string
		require.NoError(t, rows.Scan(&name, &dataType))
		cols[name] = dataType
	}
	require.NoError(t, rows.Err())

	requiredCols := []string{
		"id", "username", "avatar_url", "level", "exp",
		"rank_tier", "rank_point", "created_at", "updated_at",
	}
	for _, col := range requiredCols {
		require.Contains(t, cols, col, "expected column %q not found in users table; found: %v", col, cols)
	}

	// id must be a UUID column.
	require.Equal(t, "uuid", cols["id"], "users.id column data_type must be 'uuid'")
}

// TestMigrations_Idempotent verifies that running RunMigrations twice on the same database
// returns nil (ErrNoChange is treated as success).
func TestMigrations_Idempotent(t *testing.T) {
	dsn, cleanup := startRawPostgresContainer(t)
	defer cleanup()

	require.NoError(t, database.RunMigrations(dsn), "first RunMigrations call failed")
	require.NoError(t, database.RunMigrations(dsn), "second RunMigrations call should return nil (ErrNoChange = success)")
}
