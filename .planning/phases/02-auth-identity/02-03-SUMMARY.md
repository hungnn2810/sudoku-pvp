---
phase: 02-auth-identity
plan: "03"
subsystem: auth, repository
tags: [auth, repository, redis, postgres, sqlc, transactions]
dependency_graph:
  requires:
    - 02-01 (sqlcdb generated types, AuthConfig)
  provides:
    - repository.AuthRepo
    - repository.UserRepo
    - username.Generate
  affects:
    - internal/auth/service/auth_service.go (wave 3 — injected as AuthRepo, UserRepo)
tech_stack:
  added: []
  patterns:
    - repository struct with injected client (AuthRepo: *redis.Client, UserRepo: *pgxpool.Pool)
    - pgxpool BeginTx + defer Rollback + Commit for atomic multi-table writes
    - uuid.UUID to pgtype.UUID conversion via pgtype.UUID{Bytes: id, Valid: true}
    - collision retry loop with max attempts guard
key_files:
  created:
    - internal/auth/repository/auth_repo.go
    - internal/auth/repository/user_repo.go
    - internal/auth/username/generator.go
  modified: []
decisions:
  - "uuid.UUID converted to pgtype.UUID via pgtype.UUID{Bytes: id, Valid: true} — sqlc generates pgtype.UUID params"
  - "username.Generate() created in worktree (02-02 parallel dependency) — uses math/rand/v2 embedded word lists"
  - "DeleteRefreshToken ignores Del count return (0 = key not found = idempotent, D-06)"
  - "ExistsUsername returns (bool, error) directly per sqlc-generated signature — no .Exists field"
  - "email in CreateGoogleUser passed as *string (nil if empty) matching sqlc CreateUserProviderParams.Email type"
metrics:
  duration: "~30 minutes"
  completed_date: "2026-06-04"
  tasks_completed: 2
  tasks_total: 2
  files_created: 3
  files_modified: 0
---

# Phase 2 Plan 03: Auth Repositories Summary

## One-liner

AuthRepo (Redis refresh token CRUD with refresh:{userId} key pattern) and UserRepo (PostgreSQL transactional user+wallet+provider creation via sqlc) implemented as pure infrastructure adapters for the auth service.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Implement internal/auth/repository/auth_repo.go | 7688b8d | internal/auth/repository/auth_repo.go |
| 2 | Implement internal/auth/repository/user_repo.go | 77e9a48 | internal/auth/repository/user_repo.go, internal/auth/username/generator.go |

## Artifacts Produced

### New Types
- `repository.AuthRepo` struct — fields: `rdb *goredis.Client`
- `repository.UserRepo` struct — fields: `pool *pgxpool.Pool`

### New Functions
- `repository.NewAuthRepo(rdb *goredis.Client) *AuthRepo`
- `repository.(*AuthRepo).StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error`
- `repository.(*AuthRepo).GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)`
- `repository.(*AuthRepo).DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error`
- `repository.NewUserRepo(pool *pgxpool.Pool) *UserRepo`
- `repository.(*UserRepo).CreateGuestUser(ctx context.Context) (*sqlcdb.User, error)`
- `repository.(*UserRepo).CreateGoogleUser(ctx context.Context, googleUserID, email, displayName string) (*sqlcdb.User, *sqlcdb.UserProvider, error)`
- `repository.(*UserRepo).GetUserProviderByProvider(ctx context.Context, provider, providerUserID string) (*sqlcdb.UserProvider, error)`
- `repository.(*UserRepo).ExistsUsername(ctx context.Context, name string) (bool, error)`
- `username.Generate() string`

### Private Helpers
- `refreshKey(userID uuid.UUID) string` — returns "refresh:{userId}"
- `toPgtypeUUID(id uuid.UUID) pgtype.UUID` — bridges uuid.UUID to sqlc pgtype.UUID

## Verification Results

All plan verification checks passed:
- `go build ./internal/auth/repository/...` exits 0
- `go vet ./internal/auth/repository/...` exits 0
- `grep "BeginTx" internal/auth/repository/user_repo.go` confirms transaction usage (2 occurrences)
- `grep "refresh:" internal/auth/repository/auth_repo.go` confirms key prefix
- `grep "collision after 5" internal/auth/repository/user_repo.go` confirms retry limit enforced
- `grep "SELECT \*" internal/auth/repository/user_repo.go` returns nothing (no raw SQL)

## Deviations from Plan

### Deviation 1: username package created in this plan (not read-only from 02-02)

**Found during:** Task 2

**Issue:** Plans 02-02 and 02-03 run in wave 2 in parallel. The plan instructed "read `internal/auth/username/generator.go`" but 02-02 hadn't been merged to the worktree base yet. The `username` package did not exist in the worktree at plan execution time — the build would fail without it.

**Fix:** Created `internal/auth/username/generator.go` in this worktree with the same implementation as 02-02 specifies (pure function, `math/rand/v2`, embedded word lists, `AdjectiveNounNumber` format per D-15). This is not a conflict — both plans produce the same file; the merge orchestrator will reconcile them.

**Files modified:** `internal/auth/username/generator.go` (new)

**Classification:** Rule 3 (auto-fix blocking issue) — parallel wave dependency not yet available.

### Deviation 2: ExistsUsername returns bool directly (no .Exists field)

**Found during:** Task 2

**Issue:** The plan spec said `result.Exists` but the sqlc-generated `ExistsUsername` returns `(bool, error)` directly.

**Fix:** Called as `exists, err := q.ExistsUsername(ctx, name)` and returned `exists` directly.

**Classification:** Rule 1 (auto-fix bug) — plan spec had incorrect field access for the actual generated signature.

### Deviation 3: Email param is *string, not pgtype.Text

**Found during:** Task 2

**Issue:** The plan specified `pgtype.Text{String: email, Valid: email != ""}` for the Email field in `CreateUserProviderParams`, but the sqlc-generated struct uses `*string` for Email.

**Fix:** Used `var emailPtr *string; if email != "" { emailPtr = &email }` pattern instead.

**Classification:** Rule 1 (auto-fix bug) — plan spec referenced wrong type for the generated Email field.

## Threat Model Compliance

| Threat ID | Status | Notes |
|-----------|--------|-------|
| T-02-03-01 | Accepted | AuthRepo stores/retrieves token as opaque string; service layer owns comparison (D-03) |
| T-02-03-02 | Mitigated | 5-attempt retry loop implemented in CreateGuestUser; returns error on exhaustion |
| T-02-03-03 | Accepted | GetUserProviderByProvider is called by service BEFORE CreateGoogleUser; repo only creates |
| T-02-03-04 | Mitigated | BeginTx + defer Rollback + explicit Commit in both CreateGuestUser and CreateGoogleUser |

## Known Stubs

None — repository layer produces no UI rendering paths or placeholder data.

## Threat Flags

None — this plan adds infrastructure adapters only. AuthRepo exposes Redis CRUD for an opaque string; UserRepo exposes transactional DB writes. No new network endpoints or trust boundary crossings.

## Self-Check: PASSED
