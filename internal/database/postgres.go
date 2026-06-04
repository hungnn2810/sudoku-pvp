package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sudoku-pvp/internal/config"
)

// NewPool creates a configured pgxpool.Pool, pings the database, and returns
// the pool on success. The caller is responsible for calling pool.Close() on
// shutdown (Rule: Pitfall 6 — pool not closed on shutdown).
//
// The context controls the connection and ping timeout; callers should pass a
// context with an appropriate deadline (e.g. 30 s at startup).
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgxpool parse config: %w", err)
	}

	// Production-grade pool settings per RESEARCH.md Pattern 2.
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = 15 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.MaxConnLifetimeJitter = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}

	return pool, nil
}
