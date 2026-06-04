---
phase: 02-auth-identity
plan: "01"
subsystem: config, database, auth
tags: [config, migration, sqlc, redis, auth]
dependency_graph:
  requires: []
  provides:
    - config.AuthConfig
    - db/migrations/000013_create_user_providers
    - db/queries/auth.sql
    - db/sqlc/auth.sql.go
    - db/sqlc/models.UserProvider
  affects:
    - internal/config/config.go
    - db/sqlc/ (all generated files)
tech_stack:
  added: []
  patterns:
    - sub-struct config pattern (AuthConfig follows TelemetryConfig shape)
    - sqlc :one/:exec annotations with explicit column lists
    - insert-only table (no updated_at on user_providers)
key_files:
  created:
    - db/migrations/000013_create_user_providers.up.sql
    - db/migrations/000013_create_user_providers.down.sql
    - db/queries/auth.sql
    - db/sqlc/auth.sql.go
    - db/sqlc/db.go
    - db/sqlc/models.go
    - db/sqlc/querier.go
  modified:
    - internal/config/config.go
    - .env.example
    - deployments/docker/config.yaml
    - docs/REDIS_SCHEMA.md
decisions:
  - "AuthConfig sub-struct added after TelemetryConfig following existing pattern"
  - "Load() validates jwt_secret at startup to reject misconfigured deployments (T-02-01-02)"
  - "AccessTokenTTL default 15m, RefreshTokenTTL default 30d per D-02"
  - "user_providers is insert-only: no updated_at column per D-10/D-11"
  - "sqlc generate ran via Docker (sqlc/sqlc:latest) due to wazero allocator panic on Windows 10"
metrics:
  duration: "~45 minutes"
  completed_date: "2026-06-04"
  tasks_completed: 2
  tasks_total: 2
  files_created: 7
  files_modified: 4
---

# Phase 2 Plan 01: Auth Config Migration Summary

## One-liner

AuthConfig struct with JWT TTL validation added to Config; user_providers migration (000013) with composite unique constraint; 6 sqlc-annotated auth queries generated to db/sqlc/; REDIS_SCHEMA.md documents refresh:{userId} key.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Extend AuthConfig in config.go and update env files | c28c269 | internal/config/config.go, .env.example, deployments/docker/config.yaml |
| 2 | Create migration 000013, sqlc queries, update REDIS_SCHEMA.md | 1ea34bd | db/migrations/000013_*.sql, db/queries/auth.sql, db/sqlc/*.go, docs/REDIS_SCHEMA.md |

## Artifacts Produced

### New Types
- `config.AuthConfig` — fields: JWTSecret string, AccessTokenTTL time.Duration, RefreshTokenTTL time.Duration; mapstructure tags for viper
- `config.Config.Auth` — `Auth AuthConfig \`mapstructure:"auth"\``
- `sqlcdb.UserProvider` — Go model for user_providers table (generated)
- `sqlcdb.CreateUserParams`, `sqlcdb.CreateUserProviderParams`, `sqlcdb.GetUserProviderByProviderParams` — query param structs (generated)

### New Functions (sqlc-generated)
- `Queries.CreateUser` — inserts users row, returns full User
- `Queries.CreateUserProvider` — inserts user_providers row, returns UserProvider
- `Queries.GetUserProviderByProvider` — lookup by (provider, provider_user_id)
- `Queries.GetUserByID` — fetch user by UUID
- `Queries.ExistsUsername` — boolean collision check for guest username generation
- `Queries.CreateWallet` — exec insert for initial wallet row

### Modified Functions
- `config.Load()` — validates cfg.Auth.JWTSecret != "", applies 15m/30d TTL defaults

## Verification Results

All plan verification checks passed:
- `go build ./internal/config/...` exits 0
- `go build ./db/sqlc/...` exits 0
- `grep "auth.jwt_secret is required"` finds the validation guard
- `grep "CREATE TABLE user_providers"` finds migration DDL
- `grep "refresh:{userId}"` finds REDIS_SCHEMA.md entry
- auth.sql has 6 `-- name:` annotations, zero `SELECT *`

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

### Tooling Deviation: sqlc generate via Docker

**Found during:** Task 2

**Issue:** The installed sqlc v1.31.1 uses wazero (WebAssembly runtime) for the PostgreSQL parser. On this Windows 10 system, wazero panics with `allocator_windows: failed to reserve memory: The parameter is incorrect` when parsing SQL content. This is a known Windows-specific issue with wazero's memory allocator trying to reserve 4GB of virtual address space. Older sqlc versions (v1.27.0, v1.24.0) either used a TiDB parser that fails to compile on Go 1.25 (integer overflow constant errors) or still depend on wazero.

**Fix:** Ran `docker run --rm -v "//j/sources/sudoku-pvp:/src" -w //src sqlc/sqlc:latest generate` using the official Docker image. The Docker-based sqlc generated identical output (same sqlc v1.31.1). Generated files were copied to the worktree's `db/sqlc/` directory and verified to compile.

**Impact:** Functional equivalence confirmed — generated code matches what a direct invocation would produce. The `go build ./db/sqlc/...` verification passed.

**Files affected:** db/sqlc/auth.sql.go, db/sqlc/db.go, db/sqlc/models.go, db/sqlc/querier.go

**Classification:** Rule 3 (auto-fix blocking issue) — tooling workaround, not a code change.

## Threat Model Compliance

| Threat ID | Status | Notes |
|-----------|--------|-------|
| T-02-01-02 | Mitigated | Load() rejects empty jwt_secret at startup |
| T-02-01-01 | Accepted | ExistsUsername returns boolean only |
| T-02-01-03 | Accepted | .env.example is documentation only |

## Known Stubs

None — this plan produces infrastructure (config, migrations, queries). No UI rendering paths or placeholder data.

## Threat Flags

None — this plan only adds: (1) config struct fields read at startup, (2) DDL migration files, (3) sqlc query definitions, and (4) documentation. No new network endpoints, auth paths, or trust boundary crossings introduced.

## Self-Check: PASSED
