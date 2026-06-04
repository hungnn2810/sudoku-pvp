---
phase: 02-auth-identity
reviewed: 2026-06-04T00:00:00Z
depth: standard
files_reviewed: 25
files_reviewed_list:
  - .env.example
  - db/migrations/000013_create_user_providers.down.sql
  - db/migrations/000013_create_user_providers.up.sql
  - db/queries/auth.sql
  - db/sqlc/auth.sql.go
  - db/sqlc/db.go
  - db/sqlc/models.go
  - db/sqlc/querier.go
  - deployments/docker/config.yaml
  - docs/REDIS_SCHEMA.md
  - go.mod
  - internal/app/app.go
  - internal/auth/google/jwks.go
  - internal/auth/handler/auth_handler.go
  - internal/auth/integration/auth_test.go
  - internal/auth/jwt/jwt.go
  - internal/auth/jwt/jwt_test.go
  - internal/auth/repository/auth_repo.go
  - internal/auth/repository/user_repo.go
  - internal/auth/service/auth_service.go
  - internal/auth/service/auth_service_test.go
  - internal/auth/username/generator.go
  - internal/auth/username/generator_test.go
  - internal/common/response.go
  - internal/config/config.go
  - internal/middleware/auth.go
findings:
  critical: 3
  warning: 4
  info: 2
  total: 9
status: issues_found
---

# Phase 02: Auth & Identity — Code Review Report

**Reviewed:** 2026-06-04T00:00:00Z
**Depth:** standard
**Files Reviewed:** 25
**Status:** issues_found

## Summary

The auth phase implements guest login, Google ID token login, token refresh, and logout across a well-layered stack (handler → service → repository). The JWT signing/validation, refresh token storage in Redis, and DB transaction patterns are generally sound. Three critical issues require fixes before shipping: a panic-risk unsafe type assertion in the Logout handler, missing Google ID token `aud`/`iss` claim validation (accepting tokens issued for any Google OAuth client), and non-constant-time refresh token comparison enabling a timing side-channel. Four warnings address silent error suppression, an unguarded HTTP status code check in the JWKS fetcher, the hardcoded `"Player"` username for all Google signups, and an unneeded double-fetch path in the JWKS cache.

---

## Critical Issues

### CR-01: Unsafe type assertion in `Logout` handler causes panic on missing context key

**File:** `internal/auth/handler/auth_handler.go:104-105`

**Issue:** `c.Get("userId")` returns `(any, bool)`. The bool is silently discarded with `_`, then `userIDStr.(string)` performs an unconditional type assertion. If `"userId"` is absent from the Gin context for any reason (middleware ordering bug, future route refactor, or a unit test that bypasses middleware), `userIDStr` is `nil` and the assertion panics with `interface conversion: interface is nil, not string`. The Recovery middleware will catch the panic and return a 500, but the panic itself is a correctness defect that could mask real configuration errors.

**Fix:**
```go
// Replace lines 104-108 with a safe two-value assertion:
userIDStr, ok := c.Get("userId")
if !ok {
    common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "userId missing from context")
    return
}
userIDStrVal, ok := userIDStr.(string)
if !ok {
    common.Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", "userId context value is not a string")
    return
}
userID, err := uuid.Parse(userIDStrVal)
if err != nil {
    common.Error(c, http.StatusBadRequest, "INVALID_TOKEN", "invalid user id in token")
    return
}
```

---

### CR-02: Google ID token `aud` and `iss` claims not validated — tokens from any OAuth client accepted

**File:** `internal/auth/google/jwks.go:170-178`

**Issue:** `jwt.ParseWithClaims` is called without `jwt.WithAudience(clientID)` or `jwt.WithIssuer("https://accounts.google.com")` parser options. The golang-jwt/v5 library does NOT validate `aud` or `iss` unless explicitly instructed. As a result, any validly signed Google ID token — issued for any application's client_id — will pass verification. An attacker who holds a Google ID token obtained for a different service (e.g., a Google Sign-In for a different game, a phishing app, or any other Google OAuth application they control) can present it to this endpoint and log in as the corresponding Google user. This is a cross-client token injection attack described in the Google Identity documentation.

**Fix:**
1. Add `GOOGLE_CLIENT_ID` to config (env var `SUDOKU_AUTH_GOOGLE_CLIENT_ID`).
2. Pass the client ID to `JWKSCache` and use parser options:
```go
// In NewJWKSCache or VerifyGoogleIDToken, apply audience + issuer validation:
token, err := jwt.ParseWithClaims(
    idToken,
    claims,
    func(t *jwt.Token) (any, error) {
        if t.Method != jwt.SigningMethodRS256 {
            return nil, fmt.Errorf("google id token: unexpected signing method: %v", t.Header["alg"])
        }
        return pubKey, nil
    },
    jwt.WithAudience(c.clientID),                          // must match your OAuth client ID
    jwt.WithIssuer("https://accounts.google.com"),         // or "accounts.google.com"
    jwt.WithExpirationRequired(),
)
```
3. Validate that `claims.Issuer` is one of the two Google issuers after parsing.

---

### CR-03: Refresh token comparison uses `!=` (timing side-channel) instead of constant-time comparison

**File:** `internal/auth/service/auth_service.go:177`

**Issue:** The stored and provided refresh tokens are compared with `providedToken != storedToken`. Go's string comparison short-circuits on the first differing byte, leaking information about how many leading bytes matched via response time. An adversary who can make many requests and measure latency could use this to brute-force a refresh token byte-by-byte. The 32-byte (256-bit) token makes a full brute-force impractical, but the timing oracle still violates the principle of constant-time secret comparison and may be exploitable in low-latency, local network setups.

**Fix:**
```go
import "crypto/subtle"

// Replace line 177:
if subtle.ConstantTimeCompare([]byte(providedToken), []byte(storedToken)) != 1 {
    return TokenPair{}, fmt.Errorf("refresh: invalid refresh token")
}
```

---

## Warnings

### WR-01: JWKS fetch does not check HTTP response status code

**File:** `internal/auth/google/jwks.go:60-68`

**Issue:** `fetchLocked` calls `c.client.Do(req)` and immediately attempts to JSON-decode the response body without checking `resp.StatusCode`. If the Google JWKS endpoint returns 429 (rate limited), 500, or 503, the code attempts to decode the error response body as JWKS JSON. This fails with a confusing decode error message rather than a meaningful HTTP status. Worse, if the error response body happens to parse as valid JSON with an empty or partial `keys` array, `c.keys` is overwritten with an empty map, causing all subsequent Google logins to fail with "kid not found" until the next successful fetch.

**Fix:**
```go
resp, err := c.client.Do(req)
if err != nil {
    return fmt.Errorf("jwks fetch: %w", err)
}
defer resp.Body.Close()

if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("jwks fetch: unexpected status %d", resp.StatusCode)
}
```

---

### WR-02: `Logout` handler silently discards Redis errors

**File:** `internal/auth/handler/auth_handler.go:112`

**Issue:** `_ = h.svc.Logout(ctx, userID)` swallows all errors from the Redis `DEL` call. The comment justifies this as "logout is idempotent — deleting an already-deleted key is fine," but `DeleteRefreshToken` wraps every error, including real Redis connectivity failures (network timeout, client closed). A Redis failure during logout leaves the refresh token active in Redis with no indication to the caller or operator. The service layer should distinguish `redis.Nil` (already deleted, truly idempotent) from other errors.

**Fix (handler layer):**
```go
if err := h.svc.Logout(ctx, userID); err != nil {
    logger.FromCtx(ctx).Warn().Err(err).Str("userId", userID.String()).Msg("logout redis error")
    // Still return 204 to client — best-effort logout per D-06,
    // but log the operational failure.
}
c.Status(http.StatusNoContent)
```

---

### WR-03: All new Google users get the hardcoded username `"Player"` — uniqueness constraint will cause failures at scale

**File:** `internal/auth/service/auth_service.go:136`

**Issue:** `s.userRepo.CreateGoogleUser(ctx, googleSub, email, "Player")` passes the literal string `"Player"` as `displayName`, which becomes the user's `username` column value. The `users` table has no explicit unique constraint shown for `username` in the reviewed migration, but the guest flow uses a uniqueness-check-and-retry loop. For Google users, every new signup will attempt to insert the username `"Player"`, which will either collide with an existing `"Player"` row (causing a unique-constraint violation if one exists) or create many users all named `"Player"` (causing ambiguity). Neither is correct. The Google sign-in flow should generate a username using the same `username.Generate()` + retry logic used for guests, or derive it from the Google profile `name` claim.

**Fix:**
```go
// In auth_service.go GoogleLogin, generate a username the same way as guest:
// (pass "" or remove displayName from CreateGoogleUser signature, or accept the generated name)
generatedName, err := s.userRepo.GenerateUniqueUsername(ctx)
if err != nil {
    return TokenPair{}, fmt.Errorf("google login: %w", err)
}
user, _, createErr := s.userRepo.CreateGoogleUser(ctx, googleSub, email, generatedName)
```
The `GenerateUniqueUsername` helper can be extracted from the existing guest user retry loop in `user_repo.go`.

---

### WR-04: `GetUserProviderByProvider` in `user_repo.go` does not handle `pgx.ErrNoRows` at the repository layer

**File:** `internal/auth/repository/user_repo.go:149-158`

**Issue:** `GetUserProviderByProvider` wraps any error and returns it to the caller. The service layer (`auth_service.go:132`) relies on `errors.Is(err, pgx.ErrNoRows)` to distinguish "no existing Google user" from other errors. This works today because `%w` is used, but the repository layer violates the layering contract: repositories should translate storage-specific sentinel errors into domain errors (e.g., a custom `ErrNotFound`) rather than leaking `pgx.ErrNoRows` through the chain. If the underlying driver changes, or if a future refactor wraps with `%v` instead of `%w`, the service-layer check silently breaks and every new Google login is treated as a non-retryable error instead of a registration trigger.

**Fix:**
```go
// In user_repo.go, expose a domain error:
var ErrNotFound = errors.New("not found")

func (r *UserRepo) GetUserProviderByProvider(...) (*sqlcdb.UserProvider, error) {
    ...
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrNotFound
    }
    return nil, fmt.Errorf("user repo get provider: %w", err)
}

// In auth_service.go line 132:
if !errors.Is(err, repository.ErrNotFound) {
```

---

## Info

### IN-01: `deployments/docker/config.yaml` contains a weak default JWT secret committed to the repository

**File:** `deployments/docker/config.yaml:22`

**Issue:** `jwt_secret: "change-me-in-development"` is committed to the repository. Any developer who runs the service locally without setting `SUDOKU_AUTH_JWT_SECRET` will use this well-known secret, making all locally-issued JWTs forgeable by anyone who reads this file. The `.env.example` has a similar placeholder. This is accepted by the threat model (documented in the plan), but the config file should at minimum include a comment directing operators to override this value, and the `Load()` validator should enforce a minimum secret length (e.g., 32 bytes) to prevent accidental production deployment with a short or guessable secret.

**Fix:**
```go
// In config.go Load(), add after the JWTSecret empty check:
if len(cfg.Auth.JWTSecret) < 32 {
    return nil, fmt.Errorf("auth.jwt_secret must be at least 32 characters")
}
```

---

### IN-02: `username.Generate()` number range (10–99) is only 90 values — collision space is small

**File:** `internal/auth/username/generator.go:32`

**Issue:** The number suffix is `10 + rand.IntN(90)`, giving 90 possible values. The adjective list has 25 entries and the noun list has 25 entries, yielding a total namespace of `25 × 25 × 90 = 56,250` unique usernames. The retry loop allows 5 attempts. At approximately 40,000 registered users the birthday-paradox collision probability per attempt exceeds 50%, and the 5-attempt retry loop will begin failing routinely. This is not an immediate bug but will cause guest registration failures as the game scales. The number range should be extended (e.g., 100–9999) or more adjectives/nouns added.

**Fix:** Widen the number range to at least 1000 values (`rand.IntN(9000) + 1000`) to expand the namespace to ~5.6 million.

---

_Reviewed: 2026-06-04T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
