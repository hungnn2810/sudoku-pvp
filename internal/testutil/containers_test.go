//go:build integration

package testutil_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"sudoku-pvp/internal/testutil"
)

// TestContainerSetup verifies that all three shared container helpers start and accept
// connections. Each sub-test is run in parallel to reduce total wall-clock time.
func TestContainerSetup(t *testing.T) {
	ctx := context.Background()

	t.Run("postgres", func(t *testing.T) {
		t.Parallel()
		pool, cleanup := testutil.SetupPostgres(ctx, t)
		defer cleanup()

		require.NotNil(t, pool)
		require.NoError(t, pool.Ping(ctx))
	})

	t.Run("redis", func(t *testing.T) {
		t.Parallel()
		rdb, cleanup := testutil.SetupRedis(ctx, t)
		defer cleanup()

		require.NotNil(t, rdb)
		require.NoError(t, rdb.Ping(ctx).Err())
	})

	t.Run("rabbitmq", func(t *testing.T) {
		t.Parallel()
		amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
		defer cleanup()

		require.NotEmpty(t, amqpURL)
		require.Contains(t, amqpURL, "amqp://")
	})
}
