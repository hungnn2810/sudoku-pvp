---
id: 01-plan-integration-tests
phase: 1
plan: 05
type: execute
wave: 4
depends_on:
  - 01-plan-observability-server
objective: "Install testcontainers-go, implement integration test helpers, run integration test suite for Postgres + Redis + RabbitMQ, add smoke test for health endpoint, enforce lint gate"
files_modified:
  - internal/testutil/containers.go
  - internal/testutil/containers_test.go
  - internal/database/postgres_test.go
  - internal/rabbitmq/topology_test.go
  - internal/redis/keys_test.go
  - .golangci.yml
  - Makefile
requirements_addressed:
  - REQ-integration-test-tooling
  - REQ-test-coverage
  - REQ-nfr-api-latency
autonomous: true

must_haves:
  truths:
    - "testcontainers-go helpers start real Postgres, Redis, and RabbitMQ containers in tests"
    - "TestSchemaComplete asserts all 11 tables exist after running RunMigrations against a fresh test container"
    - "TestDeclareAll_TopologyComplete asserts all 5 queues + DLX bound correctly in a fresh RabbitMQ container"
    - "TestContainerSetup verifies all 3 container helpers start and accept connections"
    - "golangci-lint runs via `make lint` and enforces no fmt.Println + no handler-imports-repository rules"
    - "go test ./... -short exits 0 (unit suite only, no containers)"
    - "go test ./... -tags integration exits 0 when Docker is available"
  artifacts:
    - path: "internal/testutil/containers.go"
      provides: "Shared testcontainer helpers for all integration tests"
      exports: ["SetupPostgres", "SetupRedis", "SetupRabbitMQ"]
    - path: "internal/testutil/containers_test.go"
      provides: "TestContainerSetup smoke test"
      contains: "TestContainerSetup"
    - path: ".golangci.yml"
      provides: "Lint configuration enforcing project coding standards"
      contains: "forbidigo"
  key_links:
    - from: "internal/testutil/containers.go"
      to: "internal/database/migrate.go"
      via: "SetupPostgres calls RunMigrations to prepare test DB"
      pattern: "RunMigrations"
    - from: "internal/rabbitmq/topology_test.go"
      to: "internal/testutil/containers.go"
      via: "SetupRabbitMQ provides AMQP URL to topology test"
      pattern: "SetupRabbitMQ"
---

<objective>
Install testcontainers-go modules (postgres, redis, rabbitmq), implement the shared test helper package internal/testutil with SetupPostgres/SetupRedis/SetupRabbitMQ, wire integration tests for all three infra clients, and add a golangci-lint configuration enforcing project rules.

Purpose: The phase gate requires all integration tests to pass. This plan wires up the harness and ensures the test pyramid foundation is in place for every subsequent phase.
Output: go test ./... -tags integration exits 0 with real containers; go test ./... -short exits 0 for unit-only suite; make lint exits 0 with zero violations.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md
@J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md

<interfaces>
From plan 02 (internal/database/):
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error)
func RunMigrations(dsn string) error

From plan 03 (internal/redis/):
func NewClient(cfg config.RedisConfig) (*redis.Client, error)

From plan 03 (internal/rabbitmq/):
type Connection struct{}
func New(url string, logger zerolog.Logger) (*Connection, error)
func DeclareAll(ch *amqp.Channel) error

Exported by this plan (used by all future integration tests):

// internal/testutil/containers.go
func SetupPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, func())
func SetupRedis(ctx context.Context, t *testing.T) (*redis.Client, func())
func SetupRabbitMQ(ctx context.Context, t *testing.T) (string, func())  // returns AMQP URL
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Install testcontainers-go and implement shared test helpers</name>
  <files>
    internal/testutil/containers.go,
    internal/testutil/containers_test.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 10 testcontainers-go, Pitfall 7 Docker unavailable in CI, Standard Stack testcontainers versions)
  </read_first>
  <action>
    Install testcontainers-go modules:
    go get github.com/testcontainers/testcontainers-go@v0.42.0
    go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.42.0
    go get github.com/testcontainers/testcontainers-go/modules/redis@v0.42.0
    go get github.com/testcontainers/testcontainers-go/modules/rabbitmq@v0.42.0
    go get github.com/stretchr/testify@v1.10.0
    go mod tidy

    Create internal/testutil/containers.go with build tag //go:build integration. Package testutil.

    Imports: testcontainers "github.com/testcontainers/testcontainers-go", tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres", tcredis "github.com/testcontainers/testcontainers-go/modules/redis", tcrabbitmq "github.com/testcontainers/testcontainers-go/modules/rabbitmq", pgxpool "github.com/jackc/pgx/v5/pgxpool", redis "github.com/redis/go-redis/v9", database "sudoku-pvp/internal/database", config "sudoku-pvp/internal/config", require "github.com/stretchr/testify/require", testing, context, strings.

    Implement func SetupPostgres(ctx context.Context, t *testing.T) (*pgxpool.Pool, func()):
    t.Helper(). Call testcontainers.SkipIfProviderIsNotHealthy(t) to skip if Docker daemon unavailable.
    Run postgres container: tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("sudoku_test"), tcpostgres.WithUsername("test"), tcpostgres.WithPassword("test"), tcpostgres.BasicWaitStrategies()). require.NoError(t, err).
    Get connection string: ctr.ConnectionString(ctx, "sslmode=disable"). 
    Call database.RunMigrations(connStr) — require.NoError(t, err) — so test DB has full schema.
    Call database.NewPool(ctx, config.PostgresConfig{DSN: connStr, MaxConns: 5, MinConns: 1}) — require.NoError.
    Return pool and cleanup func: func() { require.NoError(t, testcontainers.TerminateContainer(ctr)) }.

    Implement func SetupRedis(ctx context.Context, t *testing.T) (*redis.Client, func()):
    t.Helper(). Call testcontainers.SkipIfProviderIsNotHealthy(t).
    Run redis container: tcredis.Run(ctx, "redis:8-alpine"). require.NoError.
    Get connection string: ctr.ConnectionString(ctx) — returns "redis://host:port". Strip "redis://" prefix for go-redis Addr field.
    Create client: redis.NewClient(&redis.Options{Addr: addr}). Ping to verify. require.NoError.
    Return client and cleanup func.

    Implement func SetupRabbitMQ(ctx context.Context, t *testing.T) (string, func()):
    t.Helper(). Call testcontainers.SkipIfProviderIsNotHealthy(t).
    Run rabbitmq container: tcrabbitmq.Run(ctx, "rabbitmq:3-management-alpine", tcrabbitmq.WithAdminUsername("guest"), tcrabbitmq.WithAdminPassword("guest")). require.NoError.
    Get AMQP URL: ctr.AmqpURL(ctx). require.NoError.
    Return amqpURL and cleanup func.

    Create internal/testutil/containers_test.go with build tag //go:build integration. Package testutil_test.

    Implement func TestContainerSetup(t *testing.T):
    t.Run("postgres", func(t *testing.T) { pool, cleanup := SetupPostgres(ctx, t); defer cleanup(); require.NotNil(t, pool); require.NoError(t, pool.Ping(ctx)) })
    t.Run("redis", func(t *testing.T) { rdb, cleanup := SetupRedis(ctx, t); defer cleanup(); require.NotNil(t, rdb); require.NoError(t, rdb.Ping(ctx).Err()) })
    t.Run("rabbitmq", func(t *testing.T) { amqpURL, cleanup := SetupRabbitMQ(ctx, t); defer cleanup(); require.NotEmpty(t, amqpURL); require.Contains(t, amqpURL, "amqp://") })
    Use t.Parallel() on each sub-test for speed.
  </action>
  <verify>
    <automated>go build ./internal/testutil/... exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/testutil/containers.go contains "//go:build integration"
    - internal/testutil/containers.go contains "func SetupPostgres("
    - internal/testutil/containers.go contains "func SetupRedis("
    - internal/testutil/containers.go contains "func SetupRabbitMQ("
    - internal/testutil/containers.go contains "SkipIfProviderIsNotHealthy"
    - internal/testutil/containers.go contains "RunMigrations" (SetupPostgres migrates the test DB)
    - internal/testutil/containers_test.go contains "//go:build integration"
    - internal/testutil/containers_test.go contains "TestContainerSetup"
    - go build ./internal/testutil/... exits 0
    - go.mod contains "github.com/testcontainers/testcontainers-go"
    - go.mod contains "github.com/stretchr/testify"
  </acceptance_criteria>
  <done>Shared testcontainers helpers implemented with SkipIfProviderIsNotHealthy guard; SetupPostgres migrates the test DB; all three helpers verified in TestContainerSetup.</done>
</task>

<task type="auto">
  <name>Task 2: Wire integration tests for all infra clients and add golangci-lint config</name>
  <files>
    internal/database/postgres_test.go,
    internal/rabbitmq/topology_test.go,
    internal/redis/keys_test.go,
    .golangci.yml,
    Makefile
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Phase Requirements Test Map, Wave 0 Gaps list, Validation Architecture section)
    J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md
    J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (PR checklist — no SELECT *, structured logging, no business logic in handler/repo)
  </read_first>
  <action>
    Update internal/database/postgres_test.go (created skeleton in plan 02 — complete the integration tests):
    Build tag //go:build integration. Package database_test.
    Import testutil "sudoku-pvp/internal/testutil".

    Implement TestMigrations(t *testing.T): call testutil.SetupPostgres(ctx, t) which already runs migrations internally. After setup, use the pool to query: SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name. Assert these 11 names appear in results: inventory_items, match_moves, match_players, matches, missions, shop_items, sudoku_puzzles, user_missions, users, wallet_transactions, wallets. Use require.ElementsMatch or iterate and assert each.

    Implement TestSchemaComplete(t *testing.T): same container setup. Query information_schema.table_columns for the users table — assert columns: id, username, avatar_url, level, exp, rank_tier, rank_point, created_at, updated_at exist. Assert id column data_type is 'uuid'. This verifies schema matches DATABASE_SCHEMA.md.

    Implement TestMigrations_Idempotent(t *testing.T): call SetupPostgres which runs migrations. Then call RunMigrations again on the same DSN — assert it returns nil (ErrNoChange is success).

    Update internal/rabbitmq/topology_test.go (created skeleton in plan 03 — complete the integration tests):
    Build tag //go:build integration. Package rabbitmq_test.
    Import testutil "sudoku-pvp/internal/testutil", amqp "github.com/rabbitmq/amqp091-go".

    Implement TestDeclareAll_TopologyComplete(t *testing.T):
    Call testutil.SetupRabbitMQ(ctx, t). Dial AMQP with amqp.Dial(amqpURL). Open channel. Call DeclareAll(ch).
    Verify each exchange exists via passive declare: ch.ExchangeDeclarePassive("game.events", "topic", true, false, false, false, nil) — require.NoError.
    Verify game.dlx exists: ch.ExchangeDeclarePassive("game.dlx", "direct", true, false, false, false, nil) — require.NoError.
    Verify each of 5 queues exists via passive declare: ch.QueueDeclarePassive("ranking.queue", ...) etc. Assert queue.Name returned equals expected.
    Verify game.dead.queue exists.

    Implement TestDeclareAll_Idempotent(t *testing.T): call DeclareAll on same channel twice — assert both calls return nil.

    The unit tests in internal/redis/keys_test.go were already implemented in plan 03 as pure string tests. No changes needed — these already pass without integration tag.

    Create .golangci.yml at project root. Configure the following linters to enforce CODING_STANDARDS.md:

    linters: enable: [gofmt, govet, errcheck, staticcheck, unused, forbidigo, noctx, godot].

    linters-settings:
    forbidigo:
      forbid:
        - pattern: "^fmt\\.Println$"
          msg: "Use zerolog structured logging, not fmt.Println (REQ-026)"
        - pattern: "^fmt\\.Printf$"
          msg: "Use zerolog structured logging, not fmt.Printf (REQ-026)"
        - pattern: "^fmt\\.Print$"
          msg: "Use zerolog structured logging, not fmt.Print (REQ-026)"
        - pattern: "^log\\.Println$"
          msg: "Use zerolog structured logging, not stdlib log (REQ-026)"

    issues:
    exclude-rules:
      - path: "_test.go"
        linters: [forbidigo]
      - path: "db/sqlc/"
        linters: [all]

    run:
    timeout: 5m
    skip-dirs: [db/sqlc, vendor]

    Update Makefile to fix the lint target. The existing placeholder `golangci-lint run ./...` stays. Also add:
    - fmt: gofmt -w ./internal/ ./cmd/
    - check: runs `go vet ./...` and the forbidigo pattern inline as a fallback when golangci-lint is not installed: `grep -rn "fmt.Println\|fmt.Printf\|fmt.Print" --include="*.go" internal/ cmd/ | grep -v "_test.go"` exits non-zero if matches found.

    Add Makefile target test-coverage:
    go test ./... -short -coverprofile=coverage.out -covermode=atomic -count=1
    go tool cover -func=coverage.out | tail -1

    Add Makefile target verify-no-fmt-println:
    @grep -rn --include="*.go" "fmt.Println\|fmt.Printf\|fmt.Print" internal/ cmd/ | grep -v "_test.go" | grep -v "^Binary" && echo "FAIL: fmt.Println found in production code" && exit 1 || echo "OK: no fmt.Println in production code"
  </action>
  <verify>
    <automated>go test ./... -short -count=1 exits 0 (unit suite, no containers)</automated>
  </verify>
  <acceptance_criteria>
    - internal/database/postgres_test.go contains "//go:build integration"
    - internal/database/postgres_test.go contains "TestMigrations"
    - internal/database/postgres_test.go contains "TestSchemaComplete"
    - internal/database/postgres_test.go contains "information_schema.tables"
    - internal/rabbitmq/topology_test.go contains "//go:build integration"
    - internal/rabbitmq/topology_test.go contains "TestDeclareAll_TopologyComplete"
    - internal/rabbitmq/topology_test.go contains "ExchangeDeclarePassive"
    - .golangci.yml contains "forbidigo"
    - .golangci.yml contains "fmt\\.Println"
    - Makefile contains "test-coverage"
    - Makefile contains "verify-no-fmt-println"
    - go test ./... -short -count=1 exits 0
    - go test ./internal/redis/... -count=1 exits 0 (key format unit tests)
    - go test ./internal/telemetry/... -count=1 exits 0 (bootstrap unit test)
  </acceptance_criteria>
  <done>Integration test suite complete for Postgres schema, RabbitMQ topology, and Redis keys; golangci-lint config enforces no fmt.Println; unit test suite passes with `go test ./... -short`.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| testcontainers → Docker socket | Integration tests require Docker socket access; containers are ephemeral and test-scoped |
| lint gate → CI pipeline | golangci-lint gates prevent security anti-patterns (raw credentials in source, fmt.Println leaking sensitive data) from merging |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-05-01 | DoS | testcontainers without Docker | mitigate | SkipIfProviderIsNotHealthy(t) in every helper — tests skip gracefully in environments without Docker |
| T-05-02 | Tampering | Test containers with real migrations | accept | Test containers use isolated ephemeral DB; no production data involved; fresh container per test suite |
| T-05-03 | Information Disclosure | Test DSN credentials in test output | accept | Test containers use "test"/"test" credentials with no real data; acceptable for ephemeral test isolation |
| T-05-04 | Tampering | golangci-lint bypassed | mitigate | CI must fail on lint errors; forbidigo prevents fmt.Println from reaching production code paths |
</threat_model>

<verification>
- go test ./... -short -count=1 exits 0 (no containers needed)
- go test ./internal/redis/... -run TestMatchStateKey -count=1 exits 0
- go test ./internal/telemetry/... -run TestBootstrap -count=1 exits 0
- internal/testutil/containers.go contains "SkipIfProviderIsNotHealthy"
- internal/testutil/containers.go contains "RunMigrations" (migrations run in SetupPostgres)
- internal/database/postgres_test.go contains "information_schema.tables"
- internal/rabbitmq/topology_test.go contains "ExchangeDeclarePassive"
- .golangci.yml exists and contains "forbidigo" with "fmt\\.Println" pattern
- Makefile contains "verify-no-fmt-println" target
- When Docker available: go test ./... -tags integration -count=1 exits 0
</verification>

<success_criteria>
- Unit test suite (go test ./... -short) passes without Docker
- Integration tests are properly gated with //go:build integration and SkipIfProviderIsNotHealthy
- TestSchemaComplete verifies all 11 tables exist post-migration
- TestDeclareAll_TopologyComplete verifies all 5 queues + DLX in real RabbitMQ
- golangci-lint config enforces no fmt.Println in production paths
- Phase 1 gate: go test ./... -tags integration exits 0 with Docker running
</success_criteria>

<output>
Create J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-05-SUMMARY.md when done
</output>
