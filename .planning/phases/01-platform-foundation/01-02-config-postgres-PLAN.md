---
id: 01-plan-config-postgres
phase: 1
plan: 02
type: execute
wave: 2
depends_on:
  - 01-01-project-scaffold
objective: "Wire Viper config, pgxpool connection, golang-migrate runner, and all PostgreSQL migration files per DATABASE_SCHEMA.md"
files_modified:
  - internal/config/config.go
  - internal/database/postgres.go
  - internal/database/migrate.go
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
requirements_addressed:
  - REQ-021
  - REQ-025
  - REQ-postgres-schema
autonomous: true

must_haves:
  truths:
    - "Config loads from env vars and YAML file with correct precedence (env overrides file)"
    - "pgxpool connects to Postgres and Ping succeeds"
    - "golang-migrate runs all 12 migration files in sequence, exit 0"
    - "All 11 tables from DATABASE_SCHEMA.md exist after migration"
    - "All 4 recommended indexes from DATABASE_SCHEMA.md exist after migration"
  artifacts:
    - path: "internal/config/config.go"
      provides: "Typed config struct with Viper loading"
      exports: ["Config", "Load"]
    - path: "internal/database/postgres.go"
      provides: "pgxpool connection factory"
      exports: ["NewPool"]
    - path: "internal/database/migrate.go"
      provides: "Migration runner"
      exports: ["RunMigrations"]
    - path: "db/migrations/000001_create_users.up.sql"
      provides: "users table DDL"
      contains: "CREATE TABLE users"
    - path: "db/migrations/000012_create_indexes.up.sql"
      provides: "All recommended indexes"
      contains: "idx_user_rank"
  key_links:
    - from: "internal/config/config.go"
      to: "internal/database/postgres.go"
      via: "PostgresConfig passed to NewPool"
      pattern: "PostgresConfig"
    - from: "internal/database/migrate.go"
      to: "db/migrations/"
      via: "file://db/migrations source path"
      pattern: "file://db/migrations"
---

<objective>
Install and wire Viper config loading, pgxpool connection, and golang-migrate runner. Write all 12 migration files covering every table and index from DATABASE_SCHEMA.md.

Purpose: Provide the typed config struct that all infra clients need, and ensure every required table exists in the DB after migration.
Output: internal/config/config.go, internal/database/postgres.go + migrate.go, 24 migration SQL files (12 up + 12 down).
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md
@J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md

<interfaces>
From internal/config/config.go (to be created — downstream plans depend on these exact field names):

type Config struct {
    Server    ServerConfig
    Postgres  PostgresConfig
    Redis     RedisConfig
    RabbitMQ  RabbitMQConfig
    Telemetry TelemetryConfig
    LogLevel  string
}

type PostgresConfig struct {
    DSN      string
    MaxConns int32
    MinConns int32
}

type RedisConfig struct {
    Addr     string
    Password string
    DB       int
    PoolSize int
}

type RabbitMQConfig struct {
    URL string
}

type TelemetryConfig struct {
    OTLPEndpoint string
}

type ServerConfig struct {
    Port int
}

func Load() (*Config, error)  -- used by cmd/api/main.go and internal/app/app.go
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error)  -- in internal/database/postgres.go
func RunMigrations(dsn string) error  -- in internal/database/migrate.go
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Install dependencies and implement Viper config</name>
  <files>
    internal/config/config.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 1 Config Loading, Pitfall 5 viper env unmarshal)
    J:\sources\sudoku-pvp\deployments\docker\config.yaml (default values to match)
    J:\sources\sudoku-pvp\.env.example (env var names to match)
  </read_first>
  <action>
    Install dependencies via go get from the project root:
    go get github.com/spf13/viper@v1.21.0
    go get github.com/jackc/pgx/v5@v5.9.2
    go get github.com/golang-migrate/migrate/v4@v4.19.1
    go get github.com/golang-migrate/migrate/v4/database/postgres
    go get github.com/golang-migrate/migrate/v4/source/file
    go get github.com/google/uuid@v1.6.0
    Then run: go mod tidy

    Create internal/config/config.go in package config. Define the following exported types exactly (downstream plans depend on these field names):

    ServerConfig: Port int mapstructure:"port"
    PostgresConfig: DSN string mapstructure:"dsn", MaxConns int32 mapstructure:"max_conns", MinConns int32 mapstructure:"min_conns"
    RedisConfig: Addr string mapstructure:"addr", Password string mapstructure:"password", DB int mapstructure:"db", PoolSize int mapstructure:"pool_size"
    RabbitMQConfig: URL string mapstructure:"url"
    TelemetryConfig: OTLPEndpoint string mapstructure:"otlp_endpoint"
    Config: Server ServerConfig mapstructure:"server", Postgres PostgresConfig mapstructure:"postgres", Redis RedisConfig mapstructure:"redis", RabbitMQ RabbitMQConfig mapstructure:"rabbitmq", Telemetry TelemetryConfig mapstructure:"telemetry", LogLevel string mapstructure:"log_level"

    Implement func Load() (*Config, error): create viper.New(), set config name "config" type "yaml", add config paths "./deployments/docker" and "." in that order. Set env prefix "SUDOKU" with AutomaticEnv(). Set key replacer strings.NewReplacer(".", "_") to map postgres.dsn → SUDOKU_POSTGRES_DSN (per RESEARCH.md Pitfall 5). Call ReadInConfig() ignoring error. Call v.Unmarshal(&cfg). Return cfg or error.

    Validate required fields: if cfg.Postgres.DSN is empty string after unmarshal, return error "postgres DSN is required". Same check for cfg.Redis.Addr and cfg.RabbitMQ.URL.
  </action>
  <verify>
    <automated>go build ./internal/config/... exits 0</automated>
  </verify>
  <acceptance_criteria>
    - internal/config/config.go contains "func Load() (*Config, error)"
    - internal/config/config.go contains "PostgresConfig"
    - internal/config/config.go contains "RedisConfig"
    - internal/config/config.go contains "RabbitMQConfig"
    - internal/config/config.go contains "TelemetryConfig"
    - internal/config/config.go contains "SetEnvKeyReplacer"
    - internal/config/config.go contains "AutomaticEnv"
    - go build ./internal/config/... exits 0
    - go.mod contains "github.com/spf13/viper"
    - go.mod contains "github.com/jackc/pgx/v5"
    - go.mod contains "github.com/golang-migrate/migrate/v4"
  </acceptance_criteria>
  <done>Viper config struct and Load() function implemented; all required dependencies installed in go.mod; package compiles cleanly.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Implement pgxpool factory and migration runner</name>
  <files>
    internal/database/postgres.go,
    internal/database/migrate.go,
    internal/database/postgres_test.go
  </files>
  <read_first>
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (Pattern 2 PostgreSQL Pool + Migration, Pitfall 3 Migration Dirty State, Pitfall 6 pgxpool not closed)
    J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md (all table and index definitions)
  </read_first>
  <behavior>
    - TestNewPool_Ping: Given valid DSN to running postgres container, NewPool returns non-nil pool and pool.Ping succeeds
    - TestNewPool_InvalidDSN: Given invalid DSN string, NewPool returns error containing "pgxpool"
    - TestRunMigrations_AllTables: After RunMigrations on fresh DB, querying information_schema.tables returns all 11 tables: users, wallets, wallet_transactions, sudoku_puzzles, matches, match_players, match_moves, missions, user_missions, shop_items, inventory_items, inventory_items
    - TestRunMigrations_Idempotent: Running RunMigrations twice on same DB returns nil error (ErrNoChange is treated as success)
  </behavior>
  <action>
    Create internal/database/postgres.go in package database. Import pgxpool "github.com/jackc/pgx/v5/pgxpool". Import config "sudoku-pvp/internal/config".

    Implement func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error):
    Parse config with pgxpool.ParseConfig(cfg.DSN) — return fmt.Errorf("pgxpool parse config: %w", err) on error.
    Set poolCfg.MaxConns = cfg.MaxConns, MinConns = cfg.MinConns, MaxConnLifetime = 15*time.Minute, MaxConnIdleTime = 5*time.Minute, MaxConnLifetimeJitter = 30*time.Second.
    Call pgxpool.NewWithConfig(ctx, poolCfg) — return fmt.Errorf("pgxpool new: %w", err) on error.
    Call pool.Ping(ctx) — return fmt.Errorf("pgxpool ping: %w", err) on error.
    Return pool, nil.

    Create internal/database/migrate.go in package database. Import migrate "github.com/golang-migrate/migrate/v4", _ "github.com/golang-migrate/migrate/v4/database/postgres", _ "github.com/golang-migrate/migrate/v4/source/file".

    Implement func RunMigrations(dsn string) error:
    Call migrate.New("file://db/migrations", dsn) — return fmt.Errorf("migrate new: %w", err) on error.
    Defer m.Close(). Call m.Up() — if err != nil and !errors.Is(err, migrate.ErrNoChange) return fmt.Errorf("migrate up: %w", err). Return nil.

    Create internal/database/postgres_test.go in package database_test (external test package). Use testcontainers-go postgres module to spin up a real container. Tests must be skipped with t.Skip if docker is unavailable (use build tag //go:build integration or check via testcontainers.SkipIfProviderIsNotHealthy style).

    Use build tag //go:build integration so these tests only run with `go test -tags integration`. In the test file, import "github.com/testcontainers/testcontainers-go/modules/postgres" as tcpostgres.

    For TestRunMigrations_AllTables: after RunMigrations call, open a stdlib sql.DB with the DSN and query: SELECT table_name FROM information_schema.tables WHERE table_schema='public'. Assert all 11 table names appear in results.
  </action>
  <verify>
    <automated>go build ./internal/database/... exits 0 (unit compile check, no container needed)</automated>
  </verify>
  <acceptance_criteria>
    - internal/database/postgres.go contains "func NewPool(ctx context.Context, cfg config.PostgresConfig)"
    - internal/database/postgres.go contains "pgxpool.NewWithConfig"
    - internal/database/postgres.go contains "MaxConnLifetime"
    - internal/database/migrate.go contains "func RunMigrations(dsn string) error"
    - internal/database/migrate.go contains "file://db/migrations"
    - internal/database/migrate.go contains "migrate.ErrNoChange"
    - internal/database/postgres_test.go contains "//go:build integration"
    - internal/database/postgres_test.go contains "TestRunMigrations_AllTables"
    - go build ./internal/database/... exits 0
  </acceptance_criteria>
  <done>pgxpool factory and migration runner implemented; integration tests written with build tag; package compiles without errors.</done>
</task>

<task type="auto">
  <name>Task 3: Write all 12 PostgreSQL migration files</name>
  <files>
    db/migrations/000001_create_users.up.sql,
    db/migrations/000001_create_users.down.sql,
    db/migrations/000002_create_wallets.up.sql,
    db/migrations/000002_create_wallets.down.sql,
    db/migrations/000003_create_wallet_transactions.up.sql,
    db/migrations/000003_create_wallet_transactions.down.sql,
    db/migrations/000004_create_sudoku_puzzles.up.sql,
    db/migrations/000004_create_sudoku_puzzles.down.sql,
    db/migrations/000005_create_matches.up.sql,
    db/migrations/000005_create_matches.down.sql,
    db/migrations/000006_create_match_players.up.sql,
    db/migrations/000006_create_match_players.down.sql,
    db/migrations/000007_create_match_moves.up.sql,
    db/migrations/000007_create_match_moves.down.sql,
    db/migrations/000008_create_missions.up.sql,
    db/migrations/000008_create_missions.down.sql,
    db/migrations/000009_create_user_missions.up.sql,
    db/migrations/000009_create_user_missions.down.sql,
    db/migrations/000010_create_shop_items.up.sql,
    db/migrations/000010_create_shop_items.down.sql,
    db/migrations/000011_create_inventory_items.up.sql,
    db/migrations/000011_create_inventory_items.down.sql,
    db/migrations/000012_create_indexes.up.sql,
    db/migrations/000012_create_indexes.down.sql
  </files>
  <read_first>
    J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md (exact DDL for every table and all recommended indexes)
    J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-RESEARCH.md (golang-migrate file naming convention, Pitfall 3)
  </read_first>
  <action>
    Create each migration file pair. golang-migrate requires exactly the naming pattern NNNNNN_name.up.sql / NNNNNN_name.down.sql. The up files contain CREATE TABLE statements and the down files contain the corresponding DROP TABLE statements in reverse dependency order.

    000001_create_users.up.sql: CREATE TABLE users per DATABASE_SCHEMA.md — id UUID PRIMARY KEY, username VARCHAR(50) NOT NULL UNIQUE, avatar_url TEXT, level INT NOT NULL DEFAULT 1, exp INT NOT NULL DEFAULT 0, rank_tier VARCHAR(20) NOT NULL DEFAULT 'Bronze', rank_point INT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL.
    000001_create_users.down.sql: DROP TABLE IF EXISTS users;

    000002_create_wallets.up.sql: CREATE TABLE wallets per schema — user_id UUID PRIMARY KEY REFERENCES users(id), coin_balance BIGINT NOT NULL DEFAULT 0, gem_balance BIGINT NOT NULL DEFAULT 0, updated_at TIMESTAMPTZ NOT NULL.
    000002_create_wallets.down.sql: DROP TABLE IF EXISTS wallets;

    000003_create_wallet_transactions.up.sql: CREATE TABLE wallet_transactions per schema — id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), type VARCHAR(50) NOT NULL, amount BIGINT NOT NULL, balance_before BIGINT NOT NULL, balance_after BIGINT NOT NULL, reference_type VARCHAR(50), reference_id VARCHAR(100), created_at TIMESTAMPTZ NOT NULL.
    000003_create_wallet_transactions.down.sql: DROP TABLE IF EXISTS wallet_transactions;

    000004_create_sudoku_puzzles.up.sql: CREATE TABLE sudoku_puzzles per schema — id UUID PRIMARY KEY, difficulty VARCHAR(20) NOT NULL, puzzle_grid JSONB NOT NULL, solution_grid JSONB NOT NULL, empty_count INT NOT NULL, created_at TIMESTAMPTZ NOT NULL.
    000004_create_sudoku_puzzles.down.sql: DROP TABLE IF EXISTS sudoku_puzzles;

    000005_create_matches.up.sql: CREATE TABLE matches per schema — id UUID PRIMARY KEY, mode VARCHAR(20) NOT NULL, status VARCHAR(20) NOT NULL, difficulty VARCHAR(20) NOT NULL, puzzle_id UUID NOT NULL REFERENCES sudoku_puzzles(id), stake_coin BIGINT NOT NULL, started_at TIMESTAMPTZ, ended_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL.
    000005_create_matches.down.sql: DROP TABLE IF EXISTS matches;

    000006_create_match_players.up.sql: CREATE TABLE match_players per schema — id UUID PRIMARY KEY, match_id UUID NOT NULL REFERENCES matches(id), user_id UUID NOT NULL REFERENCES users(id), score INT NOT NULL DEFAULT 0, progress INT NOT NULL DEFAULT 0, combo INT NOT NULL DEFAULT 0, wrong_count INT NOT NULL DEFAULT 0, hint_used INT NOT NULL DEFAULT 0, result VARCHAR(20), coin_change BIGINT DEFAULT 0, rank_change INT DEFAULT 0, finished_at TIMESTAMPTZ.
    000006_create_match_players.down.sql: DROP TABLE IF EXISTS match_players;

    000007_create_match_moves.up.sql: CREATE TABLE match_moves per schema — id UUID PRIMARY KEY, match_id UUID NOT NULL REFERENCES matches(id), user_id UUID NOT NULL REFERENCES users(id), row_index SMALLINT NOT NULL, col_index SMALLINT NOT NULL, value SMALLINT NOT NULL, is_correct BOOLEAN NOT NULL, score_delta INT NOT NULL, created_at TIMESTAMPTZ NOT NULL.
    000007_create_match_moves.down.sql: DROP TABLE IF EXISTS match_moves;

    000008_create_missions.up.sql: CREATE TABLE missions per schema — id UUID PRIMARY KEY, type VARCHAR(20) NOT NULL, name VARCHAR(100) NOT NULL, description TEXT, target_value INT NOT NULL, reward_coin BIGINT NOT NULL, reward_exp INT NOT NULL, is_active BOOLEAN NOT NULL.
    000008_create_missions.down.sql: DROP TABLE IF EXISTS missions;

    000009_create_user_missions.up.sql: CREATE TABLE user_missions per schema — id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), mission_id UUID NOT NULL REFERENCES missions(id), progress INT NOT NULL DEFAULT 0, claimed BOOLEAN NOT NULL DEFAULT FALSE.
    000009_create_user_missions.down.sql: DROP TABLE IF EXISTS user_missions;

    000010_create_shop_items.up.sql: CREATE TABLE shop_items per schema — id UUID PRIMARY KEY, type VARCHAR(50) NOT NULL, name VARCHAR(100) NOT NULL, coin_price BIGINT, gem_price BIGINT, metadata JSONB, is_active BOOLEAN NOT NULL.
    000010_create_shop_items.down.sql: DROP TABLE IF EXISTS shop_items;

    000011_create_inventory_items.up.sql: CREATE TABLE inventory_items per schema — id UUID PRIMARY KEY, user_id UUID NOT NULL REFERENCES users(id), item_id UUID NOT NULL REFERENCES shop_items(id), equipped BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL.
    000011_create_inventory_items.down.sql: DROP TABLE IF EXISTS inventory_items;

    000012_create_indexes.up.sql: Create the 4 recommended indexes from DATABASE_SCHEMA.md:
    CREATE INDEX idx_match_players_user ON match_players(user_id);
    CREATE INDEX idx_match_moves_match ON match_moves(match_id);
    CREATE INDEX idx_wallet_transactions_user ON wallet_transactions(user_id);
    CREATE INDEX idx_user_rank ON users(rank_tier, rank_point DESC);
    000012_create_indexes.down.sql: DROP INDEX IF EXISTS idx_match_players_user; DROP INDEX IF EXISTS idx_match_moves_match; DROP INDEX IF EXISTS idx_wallet_transactions_user; DROP INDEX IF EXISTS idx_user_rank;

    All up.sql files must not include IF NOT EXISTS (golang-migrate tracks version state; idempotency is handled by the schema_migrations table, not DDL guards). All timestamps stored as TIMESTAMPTZ (UTC-aware) per DEC-014.
  </action>
  <verify>
    <automated>ls db/migrations/*.sql shows 24 files (12 pairs)</automated>
  </verify>
  <acceptance_criteria>
    - db/migrations/ contains exactly 24 .sql files (12 .up.sql and 12 .down.sql)
    - db/migrations/000001_create_users.up.sql contains "CREATE TABLE users"
    - db/migrations/000002_create_wallets.up.sql contains "REFERENCES users(id)"
    - db/migrations/000011_create_inventory_items.up.sql contains "REFERENCES shop_items(id)"
    - db/migrations/000012_create_indexes.up.sql contains "idx_user_rank"
    - db/migrations/000012_create_indexes.up.sql contains "idx_match_players_user"
    - No migration up.sql contains "IF NOT EXISTS"
    - All timestamp columns use "TIMESTAMPTZ"
    - Files are numbered 000001 through 000012 sequentially with no gaps
  </acceptance_criteria>
  <done>All 11 tables and 4 indexes from DATABASE_SCHEMA.md covered by 12 sequential migration pairs; naming convention correct for golang-migrate.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| env vars → PostgresConfig.DSN | DB credentials enter via env var SUDOKU_POSTGRES_DSN; never in YAML files |
| Migration runner → PostgreSQL | DDL commands run at startup; migration user should have DDL privileges only during startup |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-01 | Information Disclosure | config.yaml / .env.example | accept | No real credentials in committed files; real secrets via env vars only per 12-factor |
| T-02-02 | Tampering | sqlc generated queries | mitigate | sqlc generates parameterized queries; no string interpolation; SELECT * forbidden per DEC-014 |
| T-02-03 | Privilege Escalation | migration DB user | accept | Dev env uses single user; production recommendation documented: separate migration user with DDL rights from app runtime user |
| T-02-04 | DoS | pgxpool blocking on unreachable DB | mitigate | NewPool wraps connection with context timeout from caller; RunMigrations should be called with context.WithTimeout(30s) at startup |
| T-01-SC | Tampering | go get installs | mitigate | All packages in Package Legitimacy Audit — golang-migrate, pgx, viper all Approved |
</threat_model>

<verification>
- go build ./internal/config/... exits 0
- go build ./internal/database/... exits 0
- internal/config/config.go contains "func Load() (*Config, error)"
- internal/database/postgres.go contains "func NewPool(ctx context.Context"
- internal/database/migrate.go contains "func RunMigrations(dsn string) error"
- 24 migration files exist in db/migrations/ (verify with: ls db/migrations/ | wc -l == 24)
- db/migrations/000012_create_indexes.up.sql contains all 4 index names
- go.mod contains github.com/spf13/viper, github.com/jackc/pgx/v5, github.com/golang-migrate/migrate/v4
</verification>

<success_criteria>
- Config package loads from env+file with correct precedence
- pgxpool factory connects to Docker-started postgres and ping succeeds
- Migration runner applies all 12 files without error to fresh database
- All 11 tables and 4 indexes exist after migration
- No secrets in committed files
</success_criteria>

<output>
Create J:\sources\sudoku-pvp\.planning\phases\01-platform-foundation\01-02-SUMMARY.md when done
</output>
