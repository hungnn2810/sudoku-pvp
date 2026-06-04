---
phase: 02-auth-identity
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/config/config.go
  - .env.example
  - deployments/docker/config.yaml
  - db/migrations/000013_create_user_providers.up.sql
  - db/migrations/000013_create_user_providers.down.sql
  - db/queries/auth.sql
  - docs/REDIS_SCHEMA.md
autonomous: true
requirements:
  - REQ-001
  - REQ-auth-guest
  - REQ-auth-google
  - REQ-auth-refresh
  - REQ-022

must_haves:
  truths:
    - "Config struct has an AuthConfig sub-struct with JWTSecret, AccessTokenTTL, RefreshTokenTTL and validation guards"
    - "Migration 000013 creates user_providers table with UUID PK, FK to users(id), composite unique on (provider, provider_user_id), UTC timestamps"
    - "db/queries/auth.sql defines all sqlc queries for auth flows: CreateUser, CreateUserProvider, GetUserProviderByProvider, GetUserByID, ExistsUsername, CreateWallet"
    - "docs/REDIS_SCHEMA.md documents refresh:{userId} key with 30-day TTL"
    - ".env.example and deployments/docker/config.yaml have SUDOKU_AUTH_* entries"
  artifacts:
    - path: "internal/config/config.go"
      provides: "AuthConfig struct + Auth field on Config + Load() validation"
      contains: "AuthConfig"
    - path: "db/migrations/000013_create_user_providers.up.sql"
      provides: "user_providers table DDL"
      contains: "CREATE TABLE user_providers"
    - path: "db/queries/auth.sql"
      provides: "sqlc-annotated queries for auth domain"
      contains: "name: CreateUser"
  key_links:
    - from: "internal/config/config.go"
      to: "internal/app/app.go"
      via: "Config struct field Auth AuthConfig"
      pattern: "Auth.*AuthConfig"
    - from: "db/queries/auth.sql"
      to: "db/sqlc/ (generated)"
      via: "sqlc generate"
      pattern: "-- name:"
---

<objective>
Lay the foundation for Phase 2: extend Config with auth settings, create the user_providers migration (000013), write all sqlc query definitions for the auth domain, document the new refresh token Redis key, and update environment files.

Purpose: Downstream plans (JWT service, repositories, service, handler) all depend on AuthConfig being available in config.Config, the user_providers table existing in migrations, and sqlc-generated types being usable from db/sqlc/. Nothing in this plan requires external HTTP calls or running services — it is pure file authoring.

Output: AuthConfig in config.go; migration files 000013 up/down; db/queries/auth.sql with 6 annotated queries; REDIS_SCHEMA.md updated; .env.example and config.yaml updated with auth section.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\.planning\ROADMAP.md
@J:\sources\sudoku-pvp\.planning\STATE.md
@J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md
@J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Extend AuthConfig in config.go and update env files</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\config\config.go (full file — understand existing sub-struct pattern and Load() validation block before touching anything)
    - J:\sources\sudoku-pvp\.env.example (current env keys)
    - J:\sources\sudoku-pvp\deployments\docker\config.yaml (current YAML structure)
  </read_first>
  <files>internal/config/config.go, .env.example, deployments/docker/config.yaml</files>
  <action>
In internal/config/config.go:

1. Add import "time" to the import block (it is not currently imported).

2. Add a new AuthConfig sub-struct after TelemetryConfig (per D-04, D-02). Fields with mapstructure tags:
   - JWTSecret string `mapstructure:"jwt_secret"`
   - AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`
   - RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`

3. Add the Auth field to the Config struct after the Telemetry field:
   Auth AuthConfig `mapstructure:"auth"`

4. In the Load() function, add validation after the existing cfg.Server.Port check. Append these guards in order:
   - if cfg.Auth.JWTSecret == "" { return nil, fmt.Errorf("auth.jwt_secret is required") }
   - if cfg.Auth.AccessTokenTTL <= 0 { cfg.Auth.AccessTokenTTL = 15 * time.Minute }  (D-02: 15min access token TTL default)
   - if cfg.Auth.RefreshTokenTTL <= 0 { cfg.Auth.RefreshTokenTTL = 30 * 24 * time.Hour }  (D-02: 30-day refresh TTL default)

In .env.example: append three lines at the end of the file (per D-04, SUDOKU_ prefix convention):
   SUDOKU_AUTH_JWT_SECRET=change-me-in-production
   SUDOKU_AUTH_ACCESS_TOKEN_TTL=15m
   SUDOKU_AUTH_REFRESH_TOKEN_TTL=720h

In deployments/docker/config.yaml: append an auth section at the bottom (per YAML nesting pattern used by all other sections):
   auth:
     jwt_secret: "change-me-in-development"
     access_token_ttl: "15m"
     refresh_token_ttl: "720h"

Do NOT change the existing SetEnvKeyReplacer — the "_" to "." replacement already handles SUDOKU_AUTH_JWT_SECRET -> auth.jwt_secret.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/config/...</automated>
  </verify>
  <acceptance_criteria>
    - internal/config/config.go compiles without errors: `go build ./internal/config/...` exits 0
    - `grep -c "AuthConfig" internal/config/config.go` returns at least 2 (struct definition + field on Config)
    - `grep "jwt_secret" internal/config/config.go` finds the mapstructure tag
    - `grep "auth.jwt_secret is required" internal/config/config.go` finds the validation guard
    - `grep "SUDOKU_AUTH_JWT_SECRET" .env.example` finds the new entry
    - `grep "jwt_secret" deployments/docker/config.yaml` finds the new YAML key
    - `grep "15 \* time.Minute" internal/config/config.go` finds the AccessTokenTTL default
  </acceptance_criteria>
  <done>AuthConfig compiles, Load() validates jwt_secret, both env files have auth section.</done>
</task>

<task type="auto">
  <name>Task 2: Create migration 000013_create_user_providers + sqlc queries + update REDIS_SCHEMA.md</name>
  <read_first>
    - J:\sources\sudoku-pvp\db\migrations\000001_create_users.up.sql (UUID PK pattern, TIMESTAMPTZ DEFAULT NOW())
    - J:\sources\sudoku-pvp\db\migrations\000003_create_wallet_transactions.up.sql (insert-only table — no updated_at)
    - J:\sources\sudoku-pvp\db\migrations\000012_create_indexes.up.sql (index creation pattern)
    - J:\sources\sudoku-pvp\db\migrations\000001_create_users.down.sql (down migration pattern)
    - J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md (add refresh token section)
    - J:\sources\sudoku-pvp\sqlc.yaml (output package name: sqlcdb, out: db/sqlc)
  </read_first>
  <files>
    db/migrations/000013_create_user_providers.up.sql,
    db/migrations/000013_create_user_providers.down.sql,
    db/queries/auth.sql,
    docs/REDIS_SCHEMA.md
  </files>
  <action>
Create db/migrations/000013_create_user_providers.up.sql with the following DDL (per D-10, D-11; insert-only table so no updated_at; UUID PK; FK to users(id); composite unique on provider+provider_user_id):

  CREATE TABLE user_providers (
      id               UUID        PRIMARY KEY,
      user_id          UUID        NOT NULL REFERENCES users(id),
      provider         VARCHAR(50) NOT NULL,
      provider_user_id VARCHAR(255) NOT NULL,
      email            TEXT,
      created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
      CONSTRAINT uq_provider_user UNIQUE (provider, provider_user_id)
  );
  CREATE INDEX idx_user_providers_user ON user_providers(user_id);

Create db/migrations/000013_create_user_providers.down.sql:
  DROP TABLE IF EXISTS user_providers;

Create db/queries/auth.sql with six sqlc-annotated queries. Each query must use explicit column lists (no SELECT *) per DEC-014. Use the package name sqlcdb (from sqlc.yaml). The six queries are:

  1. CreateUser — INSERT users row (id, username, avatar_url, created_at, updated_at) with :exec or :one returning the full row. Fields: id UUID, username VARCHAR, avatar_url TEXT nullable. Use NOW() for timestamps. Signature label: -- name: CreateUser :one
     Columns to INSERT: id, username, created_at, updated_at
     RETURNING: id, username, avatar_url, level, exp, rank_tier, rank_point, created_at, updated_at

  2. CreateUserProvider — INSERT user_providers row returning full row. Signature: -- name: CreateUserProvider :one
     Columns to INSERT: id, user_id, provider, provider_user_id, email
     RETURNING: id, user_id, provider, provider_user_id, email, created_at

  3. GetUserProviderByProvider — SELECT by (provider, provider_user_id). Signature: -- name: GetUserProviderByProvider :one
     SELECT id, user_id, provider, provider_user_id, email, created_at FROM user_providers WHERE provider = $1 AND provider_user_id = $2

  4. GetUserByID — SELECT single user by id. Signature: -- name: GetUserByID :one
     SELECT id, username, avatar_url, level, exp, rank_tier, rank_point, created_at, updated_at FROM users WHERE id = $1

  5. ExistsUsername — Check username collision for guest generation retry loop. Signature: -- name: ExistsUsername :one
     SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)::boolean AS exists

  6. CreateWallet — INSERT wallet row for new user (both guest and Google). Signature: -- name: CreateWallet :exec
     INSERT INTO wallets (user_id, coin_balance, gem_balance, updated_at) VALUES ($1, 0, 0, NOW())

In docs/REDIS_SCHEMA.md: append a new section after the existing "Reconnect State" section (per D-01, D-02, REQ-022):

  ## Refresh Token

  Key:
  refresh:{userId}

  TTL:
  30 days (720 hours)

  Stores:
  - opaque refresh token string (single token per user, per D-07)
  - overwritten on each new login and each token rotation (D-01, D-03)

After creating db/queries/auth.sql, run sqlc generate to produce db/sqlc/ typed code:
  cd J:/sources/sudoku-pvp && sqlc generate

If sqlc is not in PATH, check if it is available via: where sqlc (Windows) or the project Makefile.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && sqlc generate && go build ./db/sqlc/...</automated>
  </verify>
  <acceptance_criteria>
    - db/migrations/000013_create_user_providers.up.sql exists and contains "CREATE TABLE user_providers" and "CONSTRAINT uq_provider_user UNIQUE"
    - db/migrations/000013_create_user_providers.down.sql exists and contains "DROP TABLE IF EXISTS user_providers"
    - db/queries/auth.sql contains all six -- name: annotations: CreateUser, CreateUserProvider, GetUserProviderByProvider, GetUserByID, ExistsUsername, CreateWallet
    - `grep -c "SELECT \*" db/queries/auth.sql` returns 0 (no SELECT *)
    - `sqlc generate` exits 0 (no SQL errors)
    - db/sqlc/ contains generated .go files (db.go, auth.sql.go or similar) after sqlc generate
    - `go build ./db/sqlc/...` exits 0
    - `grep "refresh:{userId}" docs/REDIS_SCHEMA.md` finds the new section
    - `grep "30 days" docs/REDIS_SCHEMA.md` finds the TTL note
  </acceptance_criteria>
  <done>Migration 000013 files exist; sqlc generates cleanly; REDIS_SCHEMA.md has refresh token section documented.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| config file → process | Config YAML and env vars are read at startup; a tampered config file (jwt_secret = "") is rejected by the Load() validation guard |
| migration DDL → PostgreSQL | Migration files are authored by developers and applied via golang-migrate; no user input reaches DDL |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-01-01 | Information Disclosure | db/queries/auth.sql — ExistsUsername query | accept | Query returns boolean only, no user data. Username enumeration is a Phase 7 hardening concern |
| T-02-01-02 | Tampering | internal/config/config.go — JWTSecret | mitigate | Load() guard: empty jwt_secret returns error at startup, preventing service boot with a blank secret |
| T-02-01-03 | Tampering | .env.example — default change-me-in-production value | accept | Example file documents placeholder; production deployment must override via env var; not a runtime secret |
| T-02-01-SC | Tampering | npm/pip/cargo installs | accept | No new package manager installs in this plan; only file authoring and sqlc (already present) |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go build ./internal/config/...` — Config compiles with AuthConfig
2. `cd J:/sources/sudoku-pvp && sqlc generate` — SQL parses cleanly and typed output generated
3. `cd J:/sources/sudoku-pvp && go build ./db/sqlc/...` — Generated code compiles
4. `grep "auth.jwt_secret is required" J:/sources/sudoku-pvp/internal/config/config.go` — validation present
5. `grep "CREATE TABLE user_providers" J:/sources/sudoku-pvp/db/migrations/000013_create_user_providers.up.sql` — migration DDL present
6. `grep "refresh:{userId}" J:/sources/sudoku-pvp/docs/REDIS_SCHEMA.md` — Redis key documented
</verification>

<success_criteria>
- AuthConfig struct compiles as part of config.Config
- Migration 000013 files exist with correct DDL including composite unique constraint
- db/queries/auth.sql has all 6 named queries; none use SELECT *
- sqlc generate produces db/sqlc/ output without errors
- .env.example and config.yaml have auth section
- REDIS_SCHEMA.md documents refresh:{userId} with 30-day TTL
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-01-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Types
- `config.AuthConfig` struct (internal/config/config.go) — fields: JWTSecret string, AccessTokenTTL time.Duration, RefreshTokenTTL time.Duration
- `config.Config.Auth` field (internal/config/config.go) — `Auth AuthConfig \`mapstructure:"auth"\``

### New Functions
- `config.Load()` — extended: validates cfg.Auth.JWTSecret, applies AccessTokenTTL/RefreshTokenTTL defaults

### New Files
- `db/migrations/000013_create_user_providers.up.sql` — DDL for user_providers table
- `db/migrations/000013_create_user_providers.down.sql` — DROP TABLE IF EXISTS user_providers
- `db/queries/auth.sql` — 6 sqlc-annotated queries: CreateUser, CreateUserProvider, GetUserProviderByProvider, GetUserByID, ExistsUsername, CreateWallet
- `db/sqlc/auth.sql.go` (generated) — typed Go wrappers for all 6 queries
- `db/sqlc/db.go`, `db/sqlc/models.go` (generated) — DBTX interface and model structs including UserProvider

### Modified Files
- `.env.example` — adds SUDOKU_AUTH_JWT_SECRET, SUDOKU_AUTH_ACCESS_TOKEN_TTL, SUDOKU_AUTH_REFRESH_TOKEN_TTL
- `deployments/docker/config.yaml` — adds auth.jwt_secret, auth.access_token_ttl, auth.refresh_token_ttl
- `docs/REDIS_SCHEMA.md` — adds "## Refresh Token" section (key: refresh:{userId}, TTL: 30 days)
