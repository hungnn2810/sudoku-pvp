---
phase: 01-platform-foundation
plan: "05"
subsystem: testing
tags: [testcontainers, golangci-lint, integration-tests, postgres, redis, rabbitmq, go-test]

# Dependency graph
requires:
  - phase: 01-platform-foundation/02-config-postgres
    provides: "database.RunMigrations, database.NewPool used by SetupPostgres helper"
  - phase: 01-platform-foundation/03-redis-rabbitmq
    provides: "redis.NewClient, rabbitmq.DeclareAll used by integration test helpers"
  - phase: 01-platform-foundation/04-observability-server
    provides: "telemetry.Bootstrap tested by otel_test.go"

provides:
  - "internal/testutil.SetupPostgres: shared test helper (starts container, runs migrations, returns pool)"
  - "internal/testutil.SetupRedis: shared test helper (starts container, returns redis.Client)"
  - "internal/testutil.SetupRabbitMQ: shared test helper (starts container, returns AMQP URL)"
  - "postgres_test.go: TestMigrations, TestSchemaComplete (11 tables, users columns), TestMigrations_Idempotent"
  - "topology_test.go: TestDeclareAll_TopologyComplete (5 queues, 2 exchanges, DLX), TestDeclareAll_Idempotent"
  - ".golangci.yml: forbidigo rule enforcing no fmt.Println in production paths (REQ-026)"
  - "Makefile: test-coverage, verify-no-fmt-println, fmt, check targets"

affects:
  - "all future integration tests use internal/testutil helpers as shared container lifecycle"
  - "CI lint gate uses .golangci.yml to block fmt.Println from production paths"

# Tech tracking
tech-stack:
  added:
    - "github.com/testcontainers/testcontainers-go/modules/redis@v0.42.0"
  patterns:
    - "SkipIfProviderIsNotHealthy guard: all container helpers skip gracefully without Docker"
    - "Integration build tag: all container-dependent tests gated by //go:build integration"
    - "Shared testutil package: central location for all container lifecycle management"
    - "Passive declare pattern: verify RabbitMQ topology existence without modifying it"

key-files:
  created:
    - "internal/testutil/containers.go - SetupPostgres/SetupRedis/SetupRabbitMQ helpers with SkipIfProviderIsNotHealthy"
    - "internal/testutil/containers_test.go - TestContainerSetup smoke test for all three helpers"
    - ".golangci.yml - lint config with forbidigo no-fmt.Println rule"
  modified:
    - "internal/database/postgres_test.go - TestMigrations, TestSchemaComplete, TestMigrations_Idempotent (uses testutil)"
    - "internal/rabbitmq/topology_test.go - TestDeclareAll_TopologyComplete, refactored to use testutil.SetupRabbitMQ"
    - "Makefile - added test-coverage, verify-no-fmt-println, fmt, check targets"
    - "go.mod/go.sum - added testcontainers-go/modules/redis@v0.42.0"

key-decisions:
  - "SetupPostgres calls RunMigrations internally so every consumer starts with full schema without boilerplate"
  - "postgres_test.go keeps startRawPostgresContainer for idempotency test that needs to invoke RunMigrations explicitly"
  - "topology_test.go refactored to use testutil.SetupRabbitMQ; TestDeclareAll_TopologyComplete covers all 5 queues + DLX in single test"
  - "golangci-lint godot linter excluded from final config (conflicts with structured Go doc comments); core linters sufficient"

patterns-established:
  - "Pattern: All integration tests use //go:build integration + testutil.SetupX helpers"
  - "Pattern: Container helpers call SkipIfProviderIsNotHealthy as first action (T-05-01 mitigation)"
  - "Pattern: testutil.SetupPostgres handles migrations; callers do not repeat RunMigrations setup"

requirements-completed:
  - REQ-integration-test-tooling
  - REQ-test-coverage

# Metrics
duration: 30min
completed: 2026-06-04
---

# Phase 1 Plan 05: Integration Test Harness Summary

**testcontainers-go integration test harness with SetupPostgres/Redis/RabbitMQ helpers, schema verification tests, full RabbitMQ topology verification, and golangci-lint forbidigo config enforcing no fmt.Println in production paths**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-06-04
- **Completed:** 2026-06-04
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- Created `internal/testutil` package with three shared container helpers (SetupPostgres, SetupRedis, SetupRabbitMQ) that all call SkipIfProviderIsNotHealthy to skip gracefully without Docker
- SetupPostgres automatically runs RunMigrations so test consumers get a fully-migrated schema without boilerplate; TestContainerSetup verifies all three helpers start and accept connections
- postgres_test.go: TestMigrations asserts all 11 expected tables exist post-migration; TestSchemaComplete verifies users table column set including uuid id; TestMigrations_Idempotent verifies ErrNoChange handling
- rabbitmq topology_test.go: TestDeclareAll_TopologyComplete uses passive declares to verify game.events exchange, game.dlx exchange, 5 domain queues, and game.dead.queue all exist
- .golangci.yml added with forbidigo linter blocking fmt.Println/Printf/Print in production paths (REQ-026); Makefile extended with test-coverage, verify-no-fmt-println, fmt, check targets
- Unit test suite `go test ./... -short -count=1` passes with Exit 0 (no Docker required)

## Task Commits

Each task was committed atomically:

1. **Task 1: Install testcontainers-go/modules/redis and implement shared test helpers** - `3a75cef` (feat)
2. **Task 2: Wire integration tests for all infra clients and add golangci-lint config** - `2ab4a34` (feat)

## Files Created/Modified

- `internal/testutil/containers.go` - SetupPostgres (migrates), SetupRedis, SetupRabbitMQ with SkipIfProviderIsNotHealthy guard
- `internal/testutil/containers_test.go` - TestContainerSetup smoke test with t.Parallel() sub-tests
- `internal/database/postgres_test.go` - TestMigrations, TestSchemaComplete, TestMigrations_Idempotent, TestNewPool_Ping/InvalidDSN
- `internal/rabbitmq/topology_test.go` - TestDeclareAll_TopologyComplete + all prior topology tests refactored to use testutil.SetupRabbitMQ
- `.golangci.yml` - forbidigo no-fmt.Println rule, errcheck, staticcheck, unused, noctx; test files excluded
- `Makefile` - test-coverage, verify-no-fmt-println, fmt, check targets added
- `go.mod/go.sum` - testcontainers-go/modules/redis@v0.42.0 added

## Decisions Made

- SetupPostgres calls RunMigrations internally; idempotency test uses startRawPostgresContainer (without migrations) to own the RunMigrations call lifecycle directly
- TestDeclareAll_TopologyComplete uses a fresh channel per queue in passive declare loop to avoid AMQP channel closure on error
- golangci-lint `godot` linter excluded from config (conflicts with doc-comment patterns; other linters provide sufficient coverage)
- Makefile `verify-no-fmt-println` target uses grep exit-code convention: non-zero match = failure, grep no-match exit 1 = success

## Deviations from Plan

None - plan executed exactly as written. The `stretchr/testify` dependency was already present in go.mod from a prior wave (v1.11.1 as indirect); only `testcontainers-go/modules/redis` needed to be added.

## Issues Encountered

- `go mod tidy` initially removed `testcontainers-go/modules/redis` after first install because no non-integration-tagged file referenced it yet. Re-added via `go get` after creating `containers.go`; confirmed tidy does not remove it once the integration-tagged file is present.

## User Setup Required

None - no external service configuration required. Integration tests require Docker at runtime but skip gracefully without it.

## Next Phase Readiness

- Phase 1 gate achieved: unit suite (`go test ./... -short`) passes without Docker
- Integration suite (`go test ./... -tags integration`) requires Docker; all helpers guard with SkipIfProviderIsNotHealthy
- testutil helpers ready for all subsequent phases that need integration tests
- golangci-lint config in place for CI lint gate
- No blockers

---
*Phase: 01-platform-foundation*
*Completed: 2026-06-04*

## Self-Check: PASSED

- `internal/testutil/containers.go` - created and committed in 3a75cef
- `internal/testutil/containers_test.go` - created and committed in 3a75cef
- `.golangci.yml` - created and committed in 2ab4a34
- `Makefile` - updated and committed in 2ab4a34
- `internal/database/postgres_test.go` - updated and committed in 2ab4a34
- `internal/rabbitmq/topology_test.go` - updated and committed in 2ab4a34
- Commits `3a75cef` and `2ab4a34` both exist in git log
