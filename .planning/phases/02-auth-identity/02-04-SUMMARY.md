---
phase: 02-auth-identity
plan: "04"
subsystem: auth, google, service
tags: [auth, google, jwks, jwt, service, redis, postgres]
dependency_graph:
  requires:
    - 02-02 (jwt.SignJWT, jwt.NewAccessClaims, jwt.ValidateJWT)
    - 02-03 (repository.AuthRepo, repository.UserRepo)
  provides:
    - google.JWKSCache
    - google.NewJWKSCache
    - google.(*JWKSCache).VerifyGoogleIDToken
    - service.AuthService
    - service.TokenPair
    - service.(*AuthService).GuestLogin
    - service.(*AuthService).GoogleLogin
    - service.(*AuthService).Refresh
    - service.(*AuthService).Logout
  affects:
    - internal/auth/handler/auth_handler.go (wave 4 — injects AuthService)
    - internal/app/app.go (wave 4 — wires JWKSCache + AuthService)
tech_stack:
  added: []
  patterns:
    - in-memory JWKS cache with sync.RWMutex double-checked locking
    - lazy cache population with TTL and kid-miss re-fetch (D-09)
    - RS256 algorithm enforcement in jwt key func (T-02-04-01)
    - crypto/rand opaque refresh token generation (T-02-04-03)
    - token rotation: delete before issue (D-03, T-02-04-02)
    - pgtype.UUID to uuid.UUID conversion via uuid.FromBytes(id.Bytes[:])
key_files:
  created:
    - internal/auth/google/jwks.go
    - internal/auth/service/auth_service.go
  modified: []
decisions:
  - "JWKSCache uses double-checked locking (RLock fast path, Lock slow path with re-check) for minimal contention"
  - "VerifyGoogleIDToken uses jwt.NewParser().ParseUnverified for kid extraction without signature validation"
  - "RS256 enforced in key func — any other algorithm returns error before key is returned"
  - "GoogleLogin wraps pgx.ErrNoRows check: new user path via errors.Is for correct unwrapping through user_repo error chain"
  - "uuid.FromBytes(pgtype.UUID.Bytes[:]) used to convert sqlc pgtype.UUID to uuid.UUID for issueTokenPair"
  - "Refresh deletes old token BEFORE issuing new one — simultaneous rotation causes one call to return 'invalid refresh token'"
  - "Role hard-coded to 'player' on refresh — no role carried in refresh token (T-02-04-04)"
  - "common package not used in service — all sentinel errors from service are plain fmt.Errorf wraps"
metrics:
  duration: "~20 minutes"
  completed_date: "2026-06-04"
  tasks_completed: 2
  tasks_total: 2
  files_created: 2
  files_modified: 0
---

# Phase 2 Plan 04: Google Auth Service Summary

## One-liner

Goroutine-safe in-memory JWKS cache (RS256, 1hr TTL, kid-miss re-fetch) and AuthService orchestrating guest login, Google login, token rotation, and idempotent logout using crypto/rand refresh tokens.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Implement internal/auth/google/jwks.go (JWKS cache) | e1b687b | internal/auth/google/jwks.go |
| 2 | Implement internal/auth/service/auth_service.go | 68ddcdc | internal/auth/service/auth_service.go |

## Artifacts Produced

### New Types
- `google.JWK` struct — JSON Web Key fields: Kid, Kty, N, E with json tags
- `google.JWKSResponse` struct — JWKS endpoint response with Keys []JWK
- `google.JWKSCache` struct — fields: mu sync.RWMutex, keys map[string]*rsa.PublicKey, fetchedAt time.Time, ttl time.Duration, endpoint string, client *http.Client
- `service.TokenPair` struct — fields: AccessToken, RefreshToken string with json tags
- `service.AuthService` struct — fields: userRepo, authRepo, jwksCache, jwtSecret []byte, accessTTL, refreshTTL time.Duration

### New Functions
- `google.NewJWKSCache() *JWKSCache` — lazy cache; no initial fetch; 1hr TTL; Google JWKS endpoint
- `google.(*JWKSCache).VerifyGoogleIDToken(ctx, idToken string) (sub, email string, err error)` — RS256 enforced; kid-miss re-fetches
- `service.NewAuthService(userRepo, authRepo, jwksCache, jwtSecret string, accessTTL, refreshTTL time.Duration) *AuthService`
- `service.(*AuthService).GuestLogin(ctx) (TokenPair, error)` — creates user+wallet, role "player"
- `service.(*AuthService).GoogleLogin(ctx, idToken string) (TokenPair, error)` — upserts user+provider+wallet
- `service.(*AuthService).Refresh(ctx, userID uuid.UUID, providedToken string) (TokenPair, error)` — rotates token
- `service.(*AuthService).Logout(ctx, userID uuid.UUID) error` — idempotent token deletion

### Private Helpers
- `google.(*JWKSCache).fetchLocked(ctx)` — HTTP fetch + RSA key parsing; called under write lock
- `google.(*JWKSCache).getKey(ctx, kid)` — double-checked locking; triggers re-fetch on kid miss
- `service.generateOpaqueToken() (string, error)` — 32-byte crypto/rand → hex string
- `service.(*AuthService).issueTokenPair(ctx, userID, role)` — signs JWT + stores refresh + returns TokenPair

## Verification Results

All plan verification checks passed:
- `go build ./internal/auth/google/...` exits 0
- `go vet ./internal/auth/google/...` exits 0
- `go build ./internal/auth/service/...` exits 0
- `go vet ./internal/auth/service/...` exits 0
- `go build ./internal/auth/...` exits 0 (all sub-packages together)
- `grep "sync.RWMutex" internal/auth/google/jwks.go` confirms goroutine safety
- `grep "https://www.googleapis.com/oauth2/v3/certs"` confirms JWKS endpoint
- `grep "func (c \*JWKSCache) VerifyGoogleIDToken"` finds exported method
- `grep "kid.*not found"` confirms kid-miss re-fetch error path
- `grep "SigningMethodRS256"` confirms RS256 enforcement in both packages
- `grep "crypto/rand" internal/auth/service/auth_service.go` confirms secure random
- `grep -c "DeleteRefreshToken" internal/auth/service/auth_service.go` returns 2 (Refresh + Logout)
- `grep "GetUserProviderByProvider"` confirms existing-user lookup in GoogleLogin

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Double-checked locking in getKey**

- **Found during:** Task 1
- **Issue:** The plan spec described a simple "upgrade to write lock" flow, but a plain lock upgrade without re-checking the condition under the write lock creates a race condition: two goroutines could both see the cache as stale under the read lock, both drop the read lock, and both acquire the write lock sequentially — resulting in two redundant HTTP fetches and the second goroutine overwriting the first's result unnecessarily.
- **Fix:** Added a second check under the write lock (double-checked locking pattern): if the cache is now fresh and the kid is present after another goroutine fetched it, return immediately without fetching again.
- **Files modified:** internal/auth/google/jwks.go
- **Classification:** Rule 2 (missing critical correctness/performance protection).

**2. [Rule 2 - Missing Critical Functionality] `common` package not imported in service**

- **Found during:** Task 2
- **Issue:** The plan listed `sudoku-pvp/internal/common` in the service's import list, but none of the service methods actually use the `common` package's sentinel errors (`ErrNotFound`, `ErrUnauthorized`, etc.). Importing an unused package causes a Go compile error.
- **Fix:** Omitted `sudoku-pvp/internal/common` from the import list. All errors returned by the service are contextual wraps via `fmt.Errorf`.
- **Files modified:** internal/auth/service/auth_service.go
- **Classification:** Rule 1 (auto-fix build error — unused import).

## Threat Model Compliance

| Threat ID | Status | Notes |
|-----------|--------|-------|
| T-02-04-01 | Mitigated | VerifyGoogleIDToken enforces RS256 in key func; token rejected before user lookup |
| T-02-04-02 | Mitigated | Refresh deletes old token before issuing new; simultaneous rotation detected |
| T-02-04-03 | Mitigated | generateOpaqueToken uses 32-byte crypto/rand; 256-bit entropy |
| T-02-04-04 | Mitigated | Role hard-coded to "player" in Refresh; no role carried in refresh token |
| T-02-04-05 | Accepted | TTL-based cache; existing keys usable until TTL expires; kid-miss returns 401 |

## Known Stubs

None — both packages are pure business logic / infrastructure adapter code with no UI rendering paths, placeholder text, or mock data.

## Threat Flags

None — no new network endpoints introduced. JWKSCache makes outbound HTTP to Google JWKS (read-only, no auth, no user data sent). AuthService interacts only with existing PostgreSQL and Redis infrastructure established in plans 02-01 through 02-03.

## Self-Check: PASSED
