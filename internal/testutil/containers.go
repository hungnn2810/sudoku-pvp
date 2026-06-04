//go:build integration

package testutil

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq"

	"sudoku-pvp/internal/config"
	"sudoku-pvp/internal/database"
)

// SetupPostgres starts a real PostgreSQL container, runs all migrations, and returns
// a ready-to-use connection pool and a cleanup function. The cleanup function terminates
// the container; callers must call it (typically via defer).
//
// SkipIfProviderIsNotHealthy is called so tests are skipped gracefully in environments
// without a running Docker daemon (e.g., CI without Docker socket, T-05-01).
func SetupPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctr, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("sudoku_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get postgres connection string")

	// Run all migrations so the test DB has the full schema.
	err = database.RunMigrations(connStr)
	require.NoError(t, err, "failed to run migrations on test DB")

	pool, err := database.NewPool(ctx, config.PostgresConfig{
		DSN:      connStr,
		MaxConns: 5,
		MinConns: 1,
	})
	require.NoError(t, err, "failed to create postgres pool")

	cleanup := func() {
		pool.Close()
		require.NoError(t, testcontainers.TerminateContainer(ctr), "failed to terminate postgres container")
	}

	return pool, cleanup
}

// SetupRedis starts a real Redis container and returns a ready-to-use client and a cleanup
// function. The cleanup function terminates the container.
//
// SkipIfProviderIsNotHealthy is called so tests skip gracefully without Docker (T-05-01).
func SetupRedis(ctx context.Context, t *testing.T) (*goredis.Client, func()) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctr, err := tcredis.Run(ctx, "redis:8-alpine")
	require.NoError(t, err, "failed to start redis container")

	// ConnectionString returns "redis://host:port"; strip the scheme for go-redis Addr.
	rawConn, err := ctr.ConnectionString(ctx)
	require.NoError(t, err, "failed to get redis connection string")

	addr := strings.TrimPrefix(rawConn, "redis://")

	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	err = rdb.Ping(ctx).Err()
	require.NoError(t, err, "failed to ping redis container")

	cleanup := func() {
		_ = rdb.Close()
		require.NoError(t, testcontainers.TerminateContainer(ctr), "failed to terminate redis container")
	}

	return rdb, cleanup
}

// SetupRabbitMQ starts a real RabbitMQ container and returns the AMQP URL and a cleanup
// function. The cleanup function terminates the container.
//
// SkipIfProviderIsNotHealthy is called so tests skip gracefully without Docker (T-05-01).
func SetupRabbitMQ(ctx context.Context, t *testing.T) (string, func()) {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctr, err := tcrabbitmq.Run(ctx,
		"rabbitmq:3-management-alpine",
		tcrabbitmq.WithAdminUsername("guest"),
		tcrabbitmq.WithAdminPassword("guest"),
	)
	require.NoError(t, err, "failed to start rabbitmq container")

	amqpURL, err := ctr.AmqpURL(ctx)
	require.NoError(t, err, "failed to get rabbitmq AMQP URL")

	cleanup := func() {
		require.NoError(t, testcontainers.TerminateContainer(ctr), "failed to terminate rabbitmq container")
	}

	return amqpURL, cleanup
}
