---
phase: 02-auth-identity
plan: "03"
type: execute
wave: 2
depends_on:
  - 02-01
files_modified:
  - internal/auth/repository/auth_repo.go
  - internal/auth/repository/user_repo.go
autonomous: true
requirements:
  - REQ-auth-guest
  - REQ-auth-google
  - REQ-auth-refresh
  - REQ-022

must_haves:
  truths:
    - "AuthRepo exposes StoreRefreshToken, GetRefreshToken, DeleteRefreshToken operating on refresh:{userId} key with 30-day TTL"
    - "UserRepo exposes CreateGuestUser (creates users + wallets rows in a transaction) and CreateGoogleUser (creates users + user_providers + wallets in a transaction)"
    - "UserRepo.ExistsUsername checks username collision via sqlc ExistsUsername query"
    - "UserRepo.GetUserProviderByProvider queries by (provider, provider_user_id)"
    - "All public methods are context-first: func (r *X) Method(ctx context.Context, ...) ..."
    - "No raw SQL in repository files — all queries via sqlc-generated functions"
  artifacts:
    - path: "internal/auth/repository/auth_repo.go"
      provides: "AuthRepo struct with Redis refresh token operations"
      exports: ["AuthRepo", "NewAuthRepo", "StoreRefreshToken", "GetRefreshToken", "DeleteRefreshToken"]
    - path: "internal/auth/repository/user_repo.go"
      provides: "UserRepo struct with DB user/wallet creation"
      exports: ["UserRepo", "NewUserRepo", "CreateGuestUser", "CreateGoogleUser", "GetUserProviderByProvider", "ExistsUsername"]
  key_links:
    - from: "internal/auth/repository/auth_repo.go"
      to: "internal/auth/service/auth_service.go (wave 3)"
      via: "AuthRepo injected into AuthService"
      pattern: "AuthRepo"
    - from: "internal/auth/repository/user_repo.go"
      to: "internal/auth/service/auth_service.go (wave 3)"
      via: "UserRepo injected into AuthService"
      pattern: "UserRepo"
---

<objective>
Implement the auth repository layer: AuthRepo (Redis refresh token CRUD) and UserRepo (PostgreSQL user + wallet creation using sqlc-generated types). These are pure infrastructure adapters — no business logic, no JWT operations.

Purpose: Auth service (wave 3) depends on these two repository interfaces. AuthRepo owns the refresh:{userId} Redis key lifecycle. UserRepo owns user + wallet row creation in a single DB transaction for both guest and Google flows.

Output: internal/auth/repository/auth_repo.go with StoreRefreshToken/GetRefreshToken/DeleteRefreshToken; internal/auth/repository/user_repo.go with CreateGuestUser/CreateGoogleUser/GetUserProviderByProvider/ExistsUsername.
</objective>

<execution_context>
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-CONTEXT.md
@J:\sources\sudoku-pvp\.planning\phases\02-auth-identity\02-PATTERNS.md
</execution_context>

<context>
@J:\sources\sudoku-pvp\.planning\PROJECT.md
@J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md
@J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: Implement internal/auth/repository/auth_repo.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\internal\redis\client.go (Redis client type — *redis.Client injected into repos)
    - J:\sources\sudoku-pvp\internal\redis\keys.go (key constant pattern — follow same convention for refresh key)
    - J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md (refresh:{userId} key, 30-day TTL per D-01, D-02)
  </read_first>
  <files>internal/auth/repository/auth_repo.go</files>
  <action>
Create internal/auth/repository/auth_repo.go. Package name: "repository". Imports: "context", "fmt", "time", "github.com/redis/go-redis/v9", "github.com/google/uuid".

Define AuthRepo struct:
  type AuthRepo struct {
      rdb *redis.Client
  }

Define constructor:
  func NewAuthRepo(rdb *redis.Client) *AuthRepo { return &AuthRepo{rdb: rdb} }

Define refreshKey private helper:
  func refreshKey(userID uuid.UUID) string { return "refresh:" + userID.String() }

Define StoreRefreshToken:
  func (r *AuthRepo) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error
  Implementation: r.rdb.Set(ctx, refreshKey(userID), token, ttl). Wrap error: fmt.Errorf("auth repo store refresh token: %w", err). Return nil on success.

Define GetRefreshToken:
  func (r *AuthRepo) GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error)
  Implementation: r.rdb.Get(ctx, refreshKey(userID)). On redis.Nil error: return "", fmt.Errorf("auth repo get refresh token: %w", redis.Nil). Other errors: return "", fmt.Errorf("auth repo get refresh token: %w", err). Return val on success.

Define DeleteRefreshToken:
  func (r *AuthRepo) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error
  Implementation: r.rdb.Del(ctx, refreshKey(userID)). Wrap error: fmt.Errorf("auth repo delete refresh token: %w", err). Ignore count returned by Del (0 means key not found — treat as success for idempotent logout per D-06).

All three methods are context-first (context.Context as first param, per DEC-014).
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/repository/... && go vet ./internal/auth/repository/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/repository/...` exits 0
    - `go vet ./internal/auth/repository/...` exits 0
    - `grep "func (r \*AuthRepo) StoreRefreshToken" internal/auth/repository/auth_repo.go` finds the method
    - `grep "func (r \*AuthRepo) GetRefreshToken" internal/auth/repository/auth_repo.go` finds the method
    - `grep "func (r \*AuthRepo) DeleteRefreshToken" internal/auth/repository/auth_repo.go` finds the method
    - `grep "refresh:" internal/auth/repository/auth_repo.go` confirms key prefix
    - `grep "auth repo" internal/auth/repository/auth_repo.go` confirms error wrapping prefix (at least 3 matches)
  </acceptance_criteria>
  <done>AuthRepo compiles; all three Redis methods exported with context-first signatures and wrapped errors.</done>
</task>

<task type="auto">
  <name>Task 2: Implement internal/auth/repository/user_repo.go</name>
  <read_first>
    - J:\sources\sudoku-pvp\db\sqlc\ (read generated .go files after plan 02-01 runs sqlc generate — look for CreateUser, CreateUserProvider, GetUserProviderByProvider, ExistsUsername, CreateWallet function signatures)
    - J:\sources\sudoku-pvp\internal\database\postgres.go (pgxpool.Pool pattern — pool injected, BeginTx for transactions)
    - J:\sources\sudoku-pvp\internal\auth\username\generator.go (username.Generate() used in collision retry loop)
    - J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md (users + wallets + user_providers table columns)
  </read_first>
  <files>internal/auth/repository/user_repo.go</files>
  <action>
Create internal/auth/repository/user_repo.go. Package name: "repository". Imports: "context", "fmt", "github.com/jackc/pgx/v5/pgxpool", "github.com/google/uuid", "sudoku-pvp/db/sqlc", "sudoku-pvp/internal/auth/username".

Define UserRepo struct:
  type UserRepo struct {
      pool *pgxpool.Pool
  }

Define constructor:
  func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

Define ExistsUsername:
  func (r *UserRepo) ExistsUsername(ctx context.Context, name string) (bool, error)
  Implementation: q := sqlc.New(r.pool); result, err := q.ExistsUsername(ctx, name). Wrap error: fmt.Errorf("user repo exists username: %w", err). Return result.Exists, nil on success.

Define CreateGuestUser:
  func (r *UserRepo) CreateGuestUser(ctx context.Context) (*sqlc.User, error)
  Implementation:
    1. Open pgxpool transaction: tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{}). Wrap error: fmt.Errorf("user repo guest begin tx: %w", err). Defer tx.Rollback(ctx).
    2. Generate unique username with collision retry (max 5 attempts):
       - Call username.Generate() to get candidate.
       - Call ExistsUsername(ctx, candidate) using q := sqlc.New(tx).
       - If exists: generate again. After 5 attempts without unique name: return nil, fmt.Errorf("user repo: username collision after 5 attempts").
    3. Create user row: q.CreateUser(ctx, sqlc.CreateUserParams{ID: uuid.Must(uuid.NewV7()), Username: uniqueName}).
       Wrap error: fmt.Errorf("user repo create guest user: %w", err).
    4. Create wallet row: q.CreateWallet(ctx, user.ID). Wrap error: fmt.Errorf("user repo create guest wallet: %w", err).
    5. Commit: tx.Commit(ctx). Wrap error: fmt.Errorf("user repo guest commit: %w", err).
    6. Return &user, nil.

Define CreateGoogleUser:
  func (r *UserRepo) CreateGoogleUser(ctx context.Context, googleUserID, email, displayName string) (*sqlc.User, *sqlc.UserProvider, error)
  Implementation:
    1. Open transaction (same pattern as CreateGuestUser).
    2. q := sqlc.New(tx).
    3. Create user: q.CreateUser(ctx, sqlc.CreateUserParams{ID: uuid.Must(uuid.NewV7()), Username: displayName}).
       Wrap error: fmt.Errorf("user repo create google user: %w", err).
    4. Create provider: q.CreateUserProvider(ctx, sqlc.CreateUserProviderParams{ID: uuid.Must(uuid.NewV7()), UserID: user.ID, Provider: "google", ProviderUserID: googleUserID, Email: pgtype.Text{String: email, Valid: email != ""}}).
       Wrap error: fmt.Errorf("user repo create google provider: %w", err).
    5. Create wallet: q.CreateWallet(ctx, user.ID). Wrap error: fmt.Errorf("user repo create google wallet: %w", err).
    6. Commit. Return &user, &provider, nil.

Define GetUserProviderByProvider:
  func (r *UserRepo) GetUserProviderByProvider(ctx context.Context, provider, providerUserID string) (*sqlc.UserProvider, error)
  Implementation: q := sqlc.New(r.pool); result, err := q.GetUserProviderByProvider(ctx, sqlc.GetUserProviderByProviderParams{Provider: provider, ProviderUserID: providerUserID}). Wrap error: fmt.Errorf("user repo get provider: %w", err). Return &result, nil on success.

Note: uuid.NewV7() requires "github.com/google/uuid" v1.6+ which is already in go.mod from Phase 1. Confirm with grep in go.mod before using.
  </action>
  <verify>
    <automated>cd J:/sources/sudoku-pvp && go build ./internal/auth/repository/... && go vet ./internal/auth/repository/...</automated>
  </verify>
  <acceptance_criteria>
    - `go build ./internal/auth/repository/...` exits 0
    - `go vet ./internal/auth/repository/...` exits 0
    - `grep "func (r \*UserRepo) CreateGuestUser" internal/auth/repository/user_repo.go` finds the method
    - `grep "func (r \*UserRepo) CreateGoogleUser" internal/auth/repository/user_repo.go` finds the method
    - `grep "func (r \*UserRepo) GetUserProviderByProvider" internal/auth/repository/user_repo.go` finds the method
    - `grep "func (r \*UserRepo) ExistsUsername" internal/auth/repository/user_repo.go` finds the method
    - `grep "BeginTx" internal/auth/repository/user_repo.go` confirms transaction usage for user creation
    - `grep "username.Generate" internal/auth/repository/user_repo.go` confirms collision retry loop
    - `grep "SELECT \*" internal/auth/repository/user_repo.go` returns nothing (no raw SQL)
  </acceptance_criteria>
  <done>UserRepo compiles; CreateGuestUser and CreateGoogleUser use DB transactions; all methods context-first; no raw SQL.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| service → AuthRepo | Refresh token string passed from service to Redis; no external input reaches Redis directly |
| service → UserRepo | User creation params come from validated Google token or internal guest generation; no raw client input |
| Redis → GetRefreshToken | Token stored and retrieved as opaque string; service validates match before rotating |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-02-03-01 | Spoofing | GetRefreshToken — returns stored token | mitigate | Service layer compares incoming token to stored token and rotates; AuthRepo only stores/retrieves — comparison logic in service (D-03) |
| T-02-03-02 | Tampering | CreateGuestUser — username collision | mitigate | 5-attempt retry loop; after 5 failures returns error rather than infinite loop |
| T-02-03-03 | Tampering | CreateGoogleUser — duplicate Google login | mitigate | GetUserProviderByProvider is called by service BEFORE CreateGoogleUser; repo only creates, service checks existence first |
| T-02-03-04 | Elevation of Privilege | UserRepo uses DB transaction | mitigate | BeginTx + Rollback defer + explicit Commit; partial user creation (user row without wallet) is rolled back automatically |
</threat_model>

<verification>
1. `cd J:/sources/sudoku-pvp && go build ./internal/auth/repository/...` — both files compile
2. `grep "BeginTx" internal/auth/repository/user_repo.go` — transactions used for user + wallet creation
3. `grep "refresh:" internal/auth/repository/auth_repo.go` — correct Redis key prefix
4. `grep "collision after 5" internal/auth/repository/user_repo.go` — retry limit enforced
</verification>

<success_criteria>
- internal/auth/repository/auth_repo.go compiles with StoreRefreshToken/GetRefreshToken/DeleteRefreshToken
- internal/auth/repository/user_repo.go compiles with all 4 public methods
- Both repos use context-first signatures
- UserRepo uses transactions for user + wallet atomic creation
- No raw SQL — all queries via sqlc
</success_criteria>

<output>
Create .planning/phases/02-auth-identity/02-03-SUMMARY.md when done.
</output>

## Artifacts This Phase Produces

### New Types
- `repository.AuthRepo` struct (internal/auth/repository/auth_repo.go) — fields: rdb *redis.Client
- `repository.UserRepo` struct (internal/auth/repository/user_repo.go) — fields: pool *pgxpool.Pool

### New Functions
- `repository.NewAuthRepo(rdb *redis.Client) *AuthRepo`
- `repository.(*AuthRepo).StoreRefreshToken(ctx, userID uuid.UUID, token string, ttl time.Duration) error`
- `repository.(*AuthRepo).GetRefreshToken(ctx, userID uuid.UUID) (string, error)`
- `repository.(*AuthRepo).DeleteRefreshToken(ctx, userID uuid.UUID) error`
- `repository.NewUserRepo(pool *pgxpool.Pool) *UserRepo`
- `repository.(*UserRepo).CreateGuestUser(ctx) (*sqlc.User, error)`
- `repository.(*UserRepo).CreateGoogleUser(ctx, googleUserID, email, displayName string) (*sqlc.User, *sqlc.UserProvider, error)`
- `repository.(*UserRepo).GetUserProviderByProvider(ctx, provider, providerUserID string) (*sqlc.UserProvider, error)`
- `repository.(*UserRepo).ExistsUsername(ctx, name string) (bool, error)`

### New Files
- `internal/auth/repository/auth_repo.go`
- `internal/auth/repository/user_repo.go`
