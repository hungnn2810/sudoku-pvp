# Phase 2: Auth & Identity - Pattern Map

**Mapped:** 2026-06-04
**Files analyzed:** 15 (11 new + 4 modified)
**Analogs found:** 11 / 15

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `db/migrations/000013_create_user_providers.up.sql` | migration | - | `db/migrations/000003_create_wallet_transactions.up.sql` | exact |
| `db/migrations/000013_create_user_providers.down.sql` | migration | - | `db/migrations/000001_create_users.down.sql` | exact |
| `db/queries/auth.sql` | query | CRUD | *(no queries dir yet — see No Analog section)* | none |
| `internal/auth/jwt/jwt.go` | utility | transform | `internal/redis/keys.go` (pkg-level funcs, no struct deps) | role-match |
| `internal/auth/google/jwks.go` | utility | request-response | `internal/redis/client.go` (external service client, ping-on-init pattern) | partial |
| `internal/auth/repository/auth_repo.go` | repository | CRUD | `internal/redis/client.go` | role-match |
| `internal/auth/repository/user_repo.go` | repository | CRUD | `internal/database/postgres.go` | role-match |
| `internal/auth/service/auth_service.go` | service | request-response | `internal/app/app.go` (wiring + context-first) | partial |
| `internal/auth/handler/auth_handler.go` | handler | request-response | `internal/app/app.go` healthHandler + setupRouter | partial |
| `internal/middleware/auth.go` | middleware | request-response | `internal/middleware/logger.go` | exact |
| `internal/auth/username/generator.go` | utility | transform | `internal/redis/keys.go` (pure funcs, no IO) | role-match |
| `internal/config/config.go` *(modify)* | config | - | self | exact |
| `internal/app/app.go` *(modify)* | config/wiring | - | self | exact |
| `.env.example` *(modify)* | config | - | self | exact |
| `deployments/docker/config.yaml` *(modify)* | config | - | self | exact |

---

## Pattern Assignments

### `db/migrations/000013_create_user_providers.up.sql`

**Analog:** `db/migrations/000003_create_wallet_transactions.up.sql` and `db/migrations/000001_create_users.up.sql`

**Migration DDL pattern** — follow exactly: UUID PK, FK to `users(id)`, `TIMESTAMPTZ NOT NULL DEFAULT NOW()`, no `updated_at` on insert-only tables:

```sql
-- analog: db/migrations/000001_create_users.up.sql (lines 1-11)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    ...
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- analog: db/migrations/000003_create_wallet_transactions.up.sql (lines 1-11)
-- insert-only table: no updated_at, created_at without DEFAULT (caller supplies value)
CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,
    ...
    created_at TIMESTAMPTZ NOT NULL
);
```

**Apply pattern:** `user_providers` is insert-only (audit-style). Use `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, no `updated_at`. Composite unique constraint on `(provider, provider_user_id)`. Index on `user_id` (follow `000012_create_indexes.up.sql` pattern — indexes in a separate migration block or same file).

```sql
-- Target DDL shape
CREATE TABLE user_providers (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id),
    provider    VARCHAR(50) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_provider_user UNIQUE (provider, provider_user_id)
);
CREATE INDEX idx_user_providers_user ON user_providers(user_id);
```

---

### `db/migrations/000013_create_user_providers.down.sql`

**Analog:** `db/migrations/000001_create_users.down.sql` (line 1)

```sql
-- analog: db/migrations/000001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

**Apply pattern:**
```sql
DROP TABLE IF EXISTS user_providers;
```

---

### `db/queries/auth.sql`

**Analog:** None exists yet (no `db/queries/` dir). Use sqlc annotation conventions from DATABASE_SCHEMA.md and DEC-014.

**sqlc annotation pattern** — all queries must have explicit column lists (no `SELECT *`), and use named parameters. Pattern derived from DEC-014 and sqlc docs:

```sql
-- name: GetUserProviderByProvider :one
SELECT id, user_id, provider, provider_user_id, email, created_at
FROM user_providers
WHERE provider = $1 AND provider_user_id = $2;

-- name: CreateUserProvider :one
INSERT INTO user_providers (id, user_id, provider, provider_user_id, email, created_at)
VALUES ($1, $2, $3, $4, $5, NOW())
RETURNING id, user_id, provider, provider_user_id, email, created_at;

-- name: CreateUser :one
INSERT INTO users (id, username, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
RETURNING id, username, avatar_url, level, exp, rank_tier, rank_point, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, username, avatar_url, level, exp, rank_tier, rank_point, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ExistsUsername :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)::boolean;
```

---

### `internal/auth/jwt/jwt.go`

**Analog:** `internal/redis/keys.go` — package-level pure functions, no struct receiver, explicit error returns.

**Package-level function pattern** (keys.go lines 1-8, 12-15):
```go
package redis

import (
    "fmt"
    "strings"
    "github.com/google/uuid"
)

func MatchStateKey(matchID uuid.UUID) string {
    return fmt.Sprintf("match:%s:state", matchID)
}
```

**Apply pattern for jwt.go:**
```go
package jwt

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"  // add to go.mod
    "github.com/google/uuid"
)

// Claims is the JWT payload for sudoku-pvp access tokens.
// Fields: Sub (UUID v7 userId), Role, standard iat/exp.
// D-05: no username or guest flag embedded.
type Claims struct {
    Sub  uuid.UUID `json:"sub"`
    Role string    `json:"role"`
    jwt.RegisteredClaims
}

// SignJWT signs a Claims struct with HS256 and returns the compact token string.
// D-04: algorithm fixed to HS256; secret from caller (loaded from config).
func SignJWT(claims Claims, secret []byte) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString(secret)
    if err != nil {
        return "", fmt.Errorf("jwt sign: %w", err)
    }
    return signed, nil
}

// ValidateJWT parses and validates a compact token string.
// Returns *Claims on success or a wrapped error on failure.
// D-20: shared by REST middleware and WS middleware — single source of truth.
func ValidateJWT(tokenStr string, secret []byte) (*Claims, error) {
    ...
}
```

**Error wrapping pattern** — always `fmt.Errorf("context: %w", err)` (matches `internal/database/postgres.go` lines 22, 34, 39, 45).

---

### `internal/auth/google/jwks.go`

**Analog:** `internal/redis/client.go` — external service client with connectivity check on init.

**Client init pattern** (client.go lines 15-35):
```go
func NewClient(cfg config.RedisConfig) (*goredis.Client, error) {
    rdb := goredis.NewClient(&goredis.Options{ ... })
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis ping: %w", err)
    }
    return rdb, nil
}
```

**Apply pattern for jwks.go:**
```go
package google

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// jwksCache holds in-memory cached Google public keys.
// D-09: ~1hr TTL, re-fetch on kid mismatch.
// Claude's discretion: sync.RWMutex for concurrent read access.
type jwksCache struct {
    mu        sync.RWMutex
    keys      map[string]crypto.PublicKey  // kid -> key
    fetchedAt time.Time
    ttl       time.Duration
}

// NewJWKSCache creates a cache and performs an initial fetch.
// ctx controls the initial fetch timeout.
func NewJWKSCache(ctx context.Context) (*jwksCache, error) {
    c := &jwksCache{ttl: time.Hour}
    if err := c.refresh(ctx); err != nil {
        return nil, fmt.Errorf("jwks initial fetch: %w", err)
    }
    return c, nil
}

// VerifyIDToken verifies a Google ID token signature using cached JWKS.
// On kid miss, re-fetches JWKS once before failing (D-09).
func (c *jwksCache) VerifyIDToken(ctx context.Context, idToken string) (*GoogleClaims, error) {
    ...
}
```

---

### `internal/auth/repository/auth_repo.go`

**Analog:** `internal/redis/client.go` for client injection; `internal/redis/keys.go` for key naming.

**Repository struct pattern** — inject `*redis.Client`, context-first methods:
```go
package repository

import (
    "context"
    "fmt"
    "time"

    goredis "github.com/redis/go-redis/v9"
    "github.com/google/uuid"
)

type AuthRepo struct {
    rdb *goredis.Client
}

func NewAuthRepo(rdb *goredis.Client) *AuthRepo {
    return &AuthRepo{rdb: rdb}
}

// StoreRefreshToken saves token under "refresh:{userId}" with TTL.
// D-01: single token per user; overwrites on new login.
// D-03: called after rotation — old key deleted, new key set atomically via SET.
func (r *AuthRepo) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
    key := RefreshTokenKey(userID)  // see Shared Patterns
    if err := r.rdb.Set(ctx, key, token, ttl).Err(); err != nil {
        return fmt.Errorf("auth repo store refresh token: %w", err)
    }
    return nil
}
```

**Key function** — add `RefreshTokenKey` to `internal/redis/keys.go` following the same format as existing key funcs (lines 12-15):
```go
// RefreshTokenKey returns the Redis key for a user's refresh token.
// Format: "refresh:{userId}"
// TTL: 30 days (per D-01, D-02)
func RefreshTokenKey(userID uuid.UUID) string {
    return fmt.Sprintf("refresh:%s", userID)
}
```

---

### `internal/auth/repository/user_repo.go`

**Analog:** `internal/database/postgres.go` for pool injection; sqlc-generated types for query calls.

**Repository struct with pgxpool pattern** (postgres.go lines 19-49):
```go
// NewPool creates a configured pgxpool.Pool ... caller is responsible for Close().
func NewPool(ctx context.Context, cfg config.PostgresConfig) (*pgxpool.Pool, error) {
    ...
    pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
    if err != nil {
        return nil, fmt.Errorf("pgxpool new: %w", err)
    }
    ...
}
```

**Apply pattern for user_repo.go:**
```go
package repository

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "sudoku-pvp/internal/database/sqlc"  // generated
)

type UserRepo struct {
    pool *pgxpool.Pool
    q    *sqlc.Queries
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
    return &UserRepo{pool: pool, q: sqlc.New(pool)}
}

// CreateGuestUser creates a users row + wallets row in a single transaction.
// D-14: no user_providers row for guests.
// DEC-014: only Service may initiate DB transactions — UserRepo exposes a
// transactional method that service calls; repo handles the Begin/Commit/Rollback.
func (r *UserRepo) CreateGuestUser(ctx context.Context, id uuid.UUID, username string) error {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("user repo begin tx: %w", err)
    }
    defer tx.Rollback(ctx)
    ...
    return tx.Commit(ctx)
}
```

---

### `internal/auth/service/auth_service.go`

**Analog:** `internal/app/app.go` for dependency injection pattern; context-first method signatures per DEC-014.

**Context-first service method pattern** (app.go New signature, lines 45-94):
```go
func New(cfg *config.Config) (*App, error) {
    ctx := context.Background()
    // ... sequential dependency construction, each returning (dep, err)
    // fmt.Errorf("context phrase: %w", err) on every failure
}
```

**Apply pattern for auth_service.go:**
```go
package service

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
    "sudoku-pvp/internal/auth/jwt"
    "sudoku-pvp/internal/auth/repository"
    "sudoku-pvp/internal/auth/username"
    "sudoku-pvp/internal/config"
    "sudoku-pvp/internal/logger"
)

type AuthService struct {
    userRepo *repository.UserRepo
    authRepo *repository.AuthRepo
    cfg      config.AuthConfig
}

func NewAuthService(userRepo *repository.UserRepo, authRepo *repository.AuthRepo, cfg config.AuthConfig) *AuthService {
    return &AuthService{userRepo: userRepo, authRepo: authRepo, cfg: cfg}
}

// GuestLogin creates a guest user and returns token pair.
// D-14, D-15, D-17.
func (s *AuthService) GuestLogin(ctx context.Context) (accessToken, refreshToken string, err error) {
    log := logger.FromCtx(ctx)
    ...
    log.Info().Str("userId", id.String()).Msg("guest login")
    return accessToken, refreshToken, nil
}

// GoogleLogin verifies Google ID token, upserts user, returns token pair.
// D-08, D-09, D-10, D-12.
func (s *AuthService) GoogleLogin(ctx context.Context, idToken string) (accessToken, refreshToken string, err error) { ... }

// Refresh rotates refresh token and issues new access token.
// D-03.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (newAccess, newRefresh string, err error) { ... }

// Logout deletes refresh token from Redis.
// D-06.
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error { ... }
```

**Structured logging:** always `logger.FromCtx(ctx)` — never `fmt.Println` (logger.go lines 41-51, REQ-026).

---

### `internal/auth/handler/auth_handler.go`

**Analog:** `internal/app/app.go` healthHandler (lines 123-128) and setupRouter (lines 102-120).

**Thin handler pattern** (app.go lines 123-128):
```go
func healthHandler(c *gin.Context) {
    c.JSON(http.StatusOK, map[string]string{
        "status":  "ok",
        "service": "sudoku-pvp-api",
    })
}
```

**Router group registration pattern** (app.go lines 116-119):
```go
v1 := r.Group("/api/v1")
v1.GET("/health", healthHandler)
```

**Apply pattern for auth_handler.go:**
```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "sudoku-pvp/internal/auth/service"
    "sudoku-pvp/internal/logger"
)

type AuthHandler struct {
    svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
    return &AuthHandler{svc: svc}
}

// RegisterRoutes attaches auth routes to an existing router group.
// Called from internal/app/app.go setupRouter.
func (h *AuthHandler) RegisterRoutes(v1 *gin.RouterGroup) {
    auth := v1.Group("/auth")
    auth.POST("/guest", h.GuestLogin)
    auth.POST("/google", h.GoogleLogin)
    auth.POST("/refresh", h.Refresh)
    auth.POST("/logout", h.Logout)  // D-06: requires auth middleware on this route
}

func (h *AuthHandler) GuestLogin(c *gin.Context) {
    ctx := c.Request.Context()  // CRITICAL: c.Request.Context() not c.Copy() — logger.go comment
    access, refresh, err := h.svc.GuestLogin(ctx)
    if err != nil {
        logger.FromCtx(ctx).Error().Err(err).Msg("guest login failed")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"accessToken": access, "refreshToken": refresh})
}
```

**Context propagation rule:** Always `c.Request.Context()` — documented in `internal/middleware/logger.go` line 19 comment and `internal/logger/logger.go` lines 38-41.

---

### `internal/middleware/auth.go`

**Analog:** `internal/middleware/logger.go` — exact middleware structure.

**Gin middleware pattern** (logger.go lines 21-35):
```go
func ZerologLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        logger.FromCtx(c.Request.Context()).Info(). ...
    }
}
```

**Apply pattern for auth.go:**
```go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "sudoku-pvp/internal/auth/jwt"
    "sudoku-pvp/internal/logger"
)

// JWTMiddleware validates Bearer token in Authorization header for REST routes.
// D-20: calls jwt.ValidateJWT — shared with WSJWTMiddleware.
// D-21: sets c.Set("userId", claims.Sub) and c.Set("role", claims.Role).
// CRITICAL: uses c.Request.Context() for logger (not c.Copy()) — see logger.go.
func JWTMiddleware(secret []byte) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }
        claims, err := jwt.ValidateJWT(token, secret)
        if err != nil {
            logger.FromCtx(ctx).Warn().Err(err).Msg("jwt validation failed")
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        c.Set("userId", claims.Sub)
        c.Set("role", claims.Role)
        c.Next()
    }
}

// WSJWTMiddleware validates Bearer token before WebSocket upgrade.
// D-18: Authorization header (not query param) — mobile-only clients.
// D-19: returns HTTP 401 without upgrading on failure.
func WSJWTMiddleware(secret []byte) gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := c.Request.Context()
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
            return
        }
        claims, err := jwt.ValidateJWT(token, secret)
        if err != nil {
            logger.FromCtx(ctx).Warn().Err(err).Msg("ws jwt validation failed")
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        c.Set("userId", claims.Sub)
        c.Set("role", claims.Role)
        c.Next()
    }
}

func extractBearerToken(c *gin.Context) string {
    h := c.GetHeader("Authorization")
    if !strings.HasPrefix(h, "Bearer ") {
        return ""
    }
    return strings.TrimPrefix(h, "Bearer ")
}
```

**Middleware registration order** (app.go lines 108-110): Auth middleware must be registered AFTER `middleware.Tracing()` so OTel trace context is available in auth log lines. Example for protected groups:
```go
protected := v1.Group("/")
protected.Use(middleware.JWTMiddleware(cfg.Auth.JWTSecret))
```

---

### `internal/auth/username/generator.go`

**Analog:** `internal/redis/keys.go` — pure functions, no external dependencies, `fmt.Sprintf` composition.

**Pure function pattern** (keys.go lines 12-15):
```go
func MatchStateKey(matchID uuid.UUID) string {
    return fmt.Sprintf("match:%s:state", matchID)
}
```

**Apply pattern for generator.go:**
```go
package username

import (
    "fmt"
    "math/rand/v2"
)

// adjectives and nouns are embedded word lists (D-15: no external library).
var adjectives = []string{"Swift", "Bold", "Calm", "Dark", "Epic", ...}
var nouns      = []string{"King", "Wolf", "Fox",  "Bear", "Hawk", ...}

// Generate returns a random "AdjectiveNounNumber" username.
// D-15: format e.g. "SwiftKing42". Number range 10-99 for brevity.
func Generate() string {
    adj := adjectives[rand.IntN(len(adjectives))]
    noun := nouns[rand.IntN(len(nouns))]
    num := 10 + rand.IntN(90)
    return fmt.Sprintf("%s%s%d", adj, noun, num)
}
```

**Collision retry** lives in `UserRepo.CreateGuestUser` — call `username.Generate()` in a loop (max N attempts), return error if all collide. Keeps generator pure.

---

## Modified Files

### `internal/config/config.go` (modify)

**Analog:** self — add `AuthConfig` sub-struct following the exact pattern of all existing sub-structs (lines 10-39).

**Sub-struct pattern** (config.go lines 10-28):
```go
type ServerConfig struct {
    Port int `mapstructure:"port"`
}

type RedisConfig struct {
    Addr     string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB       int    `mapstructure:"db"`
    PoolSize int    `mapstructure:"pool_size"`
}
```

**Add:**
```go
// AuthConfig holds JWT signing and token TTL settings.
type AuthConfig struct {
    JWTSecret       string        `mapstructure:"jwt_secret"`
    AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
    RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}
```

Add field to `Config` struct (line 42-48 area):
```go
type Config struct {
    Server    ServerConfig    `mapstructure:"server"`
    Postgres  PostgresConfig  `mapstructure:"postgres"`
    Redis     RedisConfig     `mapstructure:"redis"`
    RabbitMQ  RabbitMQConfig  `mapstructure:"rabbitmq"`
    Telemetry TelemetryConfig `mapstructure:"telemetry"`
    Auth      AuthConfig      `mapstructure:"auth"`      // NEW
    LogLevel  string          `mapstructure:"log_level"`
}
```

Add validation in `Load()` after existing checks (lines 79-92 area):
```go
if cfg.Auth.JWTSecret == "" {
    return nil, fmt.Errorf("auth.jwt_secret is required")
}
if cfg.Auth.AccessTokenTTL <= 0 {
    cfg.Auth.AccessTokenTTL = 15 * time.Minute  // D-02 default
}
if cfg.Auth.RefreshTokenTTL <= 0 {
    cfg.Auth.RefreshTokenTTL = 30 * 24 * time.Hour  // D-02 default
}
```

---

### `internal/app/app.go` (modify)

**Wiring pattern** (app.go lines 82-93 — sequential dep construction, each line `x := NewX(deps)`):
```go
a := &App{ cfg: cfg, pool: pool, ... }
a.router = a.setupRouter()
```

**Add to `App` struct** (after `redisClient` field):
```go
// No new infrastructure fields needed — auth uses existing pool + redisClient.
```

**Add to `setupRouter`** — wire auth dependencies and register routes after existing middleware:
```go
// Auth wiring (after middleware setup, before route groups)
authUserRepo  := authrepo.NewUserRepo(a.pool)
authAuthRepo  := authrepo.NewAuthRepo(a.redisClient)
authSvc       := authsvc.NewAuthService(authUserRepo, authAuthRepo, a.cfg.Auth)
authHandler   := authhandler.NewAuthHandler(authSvc)
authHandler.RegisterRoutes(v1)

// Protected routes
protected := v1.Group("/")
protected.Use(middleware.JWTMiddleware([]byte(a.cfg.Auth.JWTSecret)))
// ... future protected routes registered on `protected`
```

**WS route** (add to `setupRouter`):
```go
r.GET("/ws/connect", middleware.WSJWTMiddleware([]byte(a.cfg.Auth.JWTSecret)), wsHandler)
```

---

### `.env.example` (modify)

**Pattern** (existing .env.example lines 1-11 — one var per line, `SUDOKU_` prefix, `KEY=value`):
```
SUDOKU_AUTH_JWT_SECRET=change-me-in-production
SUDOKU_AUTH_ACCESS_TOKEN_TTL=15m
SUDOKU_AUTH_REFRESH_TOKEN_TTL=720h
```

Env key mapping: `SUDOKU_AUTH_JWT_SECRET` → `auth.jwt_secret` (viper `SetEnvKeyReplacer` maps `_` to `.`).

---

### `deployments/docker/config.yaml` (modify)

**Pattern** (config.yaml lines 1-22 — YAML nested under section key):
```yaml
auth:
  jwt_secret: "change-me-in-development"
  access_token_ttl: "15m"
  refresh_token_ttl: "720h"
```

---

## Shared Patterns

### Context propagation (applies to ALL new files with `ctx`)
**Source:** `internal/middleware/logger.go` line 19 comment + `internal/logger/logger.go` lines 38-41

Always pass `c.Request.Context()` in Gin handlers — never `c.Copy()`. This preserves OTel trace IDs from `otelgin` middleware through to `logger.FromCtx(ctx)`.

```go
// CORRECT
ctx := c.Request.Context()
logger.FromCtx(ctx).Info().Msg("...")

// WRONG — breaks trace propagation
ctx := c.Copy()
```

### Structured logging (applies to all service and handler files)
**Source:** `internal/logger/logger.go` lines 41-51

```go
log := logger.FromCtx(ctx)
log.Info().Str("userId", id.String()).Msg("guest login")
log.Error().Err(err).Msg("google login failed")
```

Never use `fmt.Println` or `log.Printf` in production code (REQ-026).

### Error wrapping (applies to all repository and service files)
**Source:** `internal/database/postgres.go` lines 22, 34, 39, 45

```go
return nil, fmt.Errorf("descriptive context phrase: %w", err)
```

Every error from an external call (DB, Redis, HTTP) wrapped with `fmt.Errorf("...: %w", err)`. No bare `return err`.

### UUID v7 for PKs (applies to user_repo.go, auth service)
**Source:** PROJECT.md DEC-014, `db/migrations/000001_create_users.up.sql` line 2

```go
import "github.com/google/uuid"
id, err := uuid.NewV7()
```

`github.com/google/uuid` v1.6.0 is already in go.mod (line 8).

### Middleware registration order (applies to auth.go registration in app.go)
**Source:** `internal/app/app.go` lines 108-110

```go
r.Use(middleware.Recovery())    // 1st
r.Use(middleware.Tracing())     // 2nd — injects OTel span
r.Use(middleware.ZerologLogger()) // 3rd — reads span from context
// JWTMiddleware on route groups AFTER the above three
```

Auth middleware must be on route groups (not global), after tracing is registered globally.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `db/queries/auth.sql` | query | CRUD | No `db/queries/` directory exists yet; first sqlc query file in the project |
| `internal/auth/google/jwks.go` | utility | request-response | No HTTP client for external APIs exists yet; pattern derived from `redis/client.go` structure only |

For these files, follow the RESEARCH.md patterns and the conventions documented above. The `db/queries/auth.sql` pattern is provided in the Pattern Assignments section above.

---

## New Dependency Required

`github.com/golang-jwt/jwt/v5` must be added to `go.mod`. This is not yet present (verified against go.mod lines 1-28). Run `go get github.com/golang-jwt/jwt/v5` before implementing `internal/auth/jwt/jwt.go`.

---

## Metadata

**Analog search scope:** `internal/middleware/`, `internal/config/`, `internal/app/`, `internal/database/`, `internal/redis/`, `internal/logger/`, `db/migrations/`
**Files scanned:** 12
**Pattern extraction date:** 2026-06-04
