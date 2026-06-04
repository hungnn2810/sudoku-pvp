---
phase: 1
plan: 02
subsystem: config-postgres
tags: [config, viper, pgxpool, golang-migrate, migrations, database, tdd]
dependency_graph:
  requires: [01-01-project-scaffold]
  provides:
    - internal/config.Config (typed config struct consumed by all infra adapters)
    - internal/database.NewPool (pgxpool factory consumed by app.go)
    - internal/database.RunMigrations (migration runner called at startup)
    - db/migrations/ (12 sequential migration pairs, 11 tables + 4 indexes)
  affects:
    - cmd/api/main.go (calls Load() and RunMigrations at startup)
    - internal/app/app.go (receives *pgxpool.Pool and config.Config)
    - db/sqlc/ (sqlc reads from db/migrations for schema)
tech_stack:
  added:
    - github.com/spf13/viper v1.21.0
    - github.com/jackc/pgx/v5 v5.9.2
    - github.com/golang-migrate/migrate/v4 v4.19.1
    - github.com/golang-migrate/migrate/v4/database/postgres
    - github.com/golang-migrate/migrate/v4/source/file
    - github.com/testcontainers/testcontainers-go/modules/postgres v0.42.0
  patterns:
    - Viper with SetEnvKeyReplacer for 12-factor env var overlay
    - pgxpool.NewWithConfig with MaxConnLifetime/MaxConnIdleTime/Jitter
    - golang-migrate ErrNoChange treated as success (idempotent)
    - TDD integration tests with build tag //go:build integration
key_files:
  created:
    - internal/config/config.go
    - internal/database/postgres.go
    - internal/database/migrate.go
    - internal/database/postgres_test.go
    - db/migrations/000001_create_users.up.sql
    - db/migrations/000001_create_users.down.sql
    - db/migrations/000002_create_wallets.up.sql
    - db/migrations/000002_create_wallets.down.sql
    - db/migrations/000003_create_wallet_transactions.up.sql
    - db/migrations/000003_create_wallet_transactions.down.sql
    - db/migrations/000004_create_sudoku_puzzles.up.sql
    - db/migrations/000004_create_sudoku_puzzles.down.sql
    - db/migrations/000005_create_matches.up.sql
    - db/migrations/000005_create_matches.down.sql
    - db/migrations/000006_create_match_players.up.sql
    - db/migrations/000006_create_match_players.down.sql
    - db/migrations/000007_create_match_moves.up.sql
    - db/migrations/000007_create_match_moves.down.sql
    - db/migrations/000008_create_missions.up.sql
    - db/migrations/000008_create_missions.down.sql
    - db/migrations/000009_create_user_missions.up.sql
    - db/migrations/000009_create_user_missions.down.sql
    - db/migrations/000010_create_shop_items.up.sql
    - db/migrations/000010_create_shop_items.down.sql
    - db/migrations/000011_create_inventory_items.up.sql
    - db/migrations/000011_create_inventory_items.down.sql
    - db/migrations/000012_create_indexes.up.sql
    - db/migrations/000012_create_indexes.down.sql
  modified:
    - go.mod (added viper, pgx/v5, golang-migrate, testcontainers)
    - go.sum (generated)
decisions:
  - Validate required fields (Postgres.DSN, Redis.Addr, RabbitMQ.URL) after Unmarshal to fail fast
  - Close pool on Ping failure in NewPool to avoid connection leak
  - Use file://db/migrations source path for golang-migrate (colocation with sqlc queries)
  - Migration .gitkeep removed in favor of actual SQL files
  - Integration tests tagged with //go:build integration to avoid running without Docker
metrics:
  duration: "~8 minutes"
  completed: "2026-06-04"
  tasks_completed: 3
  files_created: 30
  files_modified: 2
---

# Phase 1 Plan 02: Config + PostgreSQL + Migrations Summary

**One-liner:** Viper config with SUDOKU_ env prefix, pgxpool factory with production pool settings, golang-migrate runner, and 24 SQL files covering all 11 tables and 4 indexes from DATABASE_SCHEMA.md.

## Tasks Completed

| # | Task | Commit | Status |
|---|------|--------|--------|
| 1 | Install dependencies and implement Viper config | c4b6651 | Done |
| 2 (RED) | Add failing integration tests for pgxpool + migration runner | 407ba04 | Done |
| 2 (GREEN) | Implement pgxpool factory and migration runner | ba0737e | Done |
| 3 | Write all 12 PostgreSQL migration file pairs | dcbe885 | Done |

## What Was Built

### internal/config/config.go
Typed `Config` struct with Viper loading. Env prefix `SUDOKU_` with `SetEnvKeyReplacer` maps nested keys (e.g., `SUDOKU_POSTGRES_DSN` → `postgres.dsn`). Reads YAML from `./deployments/docker/config.yaml` then overlays env vars. Validates required fields at startup: `Postgres.DSN`, `Redis.Addr`, `RabbitMQ.URL`.

### internal/database/postgres.go
`NewPool(ctx, cfg)` creates a pgxpool.Pool with production settings:
- MaxConns / MinConns from config
- MaxConnLifetime: 15 minutes
- MaxConnIdleTime: 5 minutes
- MaxConnLifetimeJitter: 30 seconds
- Pings on creation; closes pool and returns error on ping failure

### internal/database/migrate.go
`RunMigrations(dsn)` applies SQL files from `file://db/migrations`. Returns nil on `migrate.ErrNoChange` (idempotent). All migrations run in sequence (000001–000012).

### internal/database/postgres_test.go
Integration tests with `//go:build integration` build tag:
- `TestNewPool_Ping`: valid DSN to real container → pool non-nil, Ping succeeds
- `TestNewPool_InvalidDSN`: invalid DSN → error containing "pgxpool"
- `TestRunMigrations_AllTables`: after migration, all 11 tables in `information_schema.tables`
- `TestRunMigrations_Idempotent`: running twice returns nil

### db/migrations/ (24 files, 12 pairs)
All 11 tables from DATABASE_SCHEMA.md covered in dependency order:
1. users → 2. wallets → 3. wallet_transactions → 4. sudoku_puzzles → 5. matches → 6. match_players → 7. match_moves → 8. missions → 9. user_missions → 10. shop_items → 11. inventory_items → 12. indexes (4 recommended indexes)

## Deviations from Plan

None - plan executed exactly as written.

The only process note: the worktree was created before plan 01 commits landed on `main`. A `git merge main` fast-forward was applied first to bring the scaffold files into the worktree before executing plan 02.

## TDD Gate Compliance

| Gate | Commit | Status |
|------|--------|--------|
| RED: failing test | 407ba04 (test(01-02): add failing integration tests) | Passed |
| GREEN: implementation | ba0737e (feat(01-02): implement pgxpool factory) | Passed |
| REFACTOR | Not required | N/A |

## Known Stubs

None. All config, connection, and migration functions are fully wired and not placeholder implementations.

## Threat Flags

No new security-relevant surface beyond what was documented in the plan's threat model. All secrets remain in env vars; no credentials in committed YAML files.

## Self-Check: PASSED

Files verified:
- internal/config/config.go: exists, contains func Load() (*Config, error)
- internal/database/postgres.go: exists, contains func NewPool(ctx context.Context
- internal/database/migrate.go: exists, contains func RunMigrations(dsn string) error
- internal/database/postgres_test.go: exists, contains //go:build integration and TestRunMigrations_AllTables
- db/migrations/: 24 .sql files (000001-000012, 12 pairs)
- go.mod: contains spf13/viper, jackc/pgx/v5, golang-migrate/migrate/v4

Commits verified:
- c4b6651: feat(01-02): implement Viper config and install dependencies
- 407ba04: test(01-02): add failing integration tests for pgxpool factory and migration runner
- ba0737e: feat(01-02): implement pgxpool factory and migration runner
- dcbe885: feat(01-02): write all 12 PostgreSQL migration file pairs
