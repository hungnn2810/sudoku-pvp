---
phase: 02-auth-identity
fixed_at: 2026-06-04T00:00:00Z
review_path: .planning/phases/02-auth-identity/02-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 02: Auth & Identity — Code Review Fix Report

**Fixed at:** 2026-06-04T00:00:00Z
**Source review:** .planning/phases/02-auth-identity/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (3 Critical + 4 Warning)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: Unsafe type assertion in `Logout` handler causes panic on missing context key

**Files modified:** `internal/auth/handler/auth_handler.go`
**Commit:** `1f85792`
**Applied fix:** Replaced the unconditional `userIDStr.(string)` type assertion with a two-value safe assertion pattern. Added `ok` checks for both `c.Get("userId")` (missing key) and the subsequent `.(string)` cast. Returns HTTP 500 with a meaningful error code for each failure path. Also incorporated WR-02 fix in the same handler: replaced `_ = h.svc.Logout(...)` with an `if err != nil` block that logs Redis errors at Warn level while still returning 204 to the client.

---

### CR-02: Google ID token `aud` and `iss` claims not validated

**Files modified:** `internal/auth/google/jwks.go`, `internal/config/config.go`, `internal/app/app.go`, `internal/auth/integration/auth_test.go`, `deployments/docker/config.yaml`, `.env.example`
**Commit:** `c5c345d`
**Applied fix:** Added `GoogleClientID string` field to `AuthConfig` (env var `SUDOKU_AUTH_GOOGLE_CLIENT_ID`). Made it a required field with a validation error in `config.Load()`. Added `clientID string` field to `JWKSCache` struct; changed `NewJWKSCache` signature to `NewJWKSCache(clientID string)`. Passed `jwt.WithAudience(c.clientID)`, `jwt.WithIssuer("https://accounts.google.com")`, and `jwt.WithExpirationRequired()` as parser options to `jwt.ParseWithClaims` in `VerifyGoogleIDToken`. Updated `app.go` to pass `cfg.Auth.GoogleClientID`. Updated integration test config to include a `GoogleClientID` value. Added placeholder to `deployments/docker/config.yaml` and `SUDOKU_AUTH_GOOGLE_CLIENT_ID` to `.env.example`.

---

### CR-03: Refresh token comparison uses `!=` (timing side-channel)

**Files modified:** `internal/auth/service/auth_service.go`
**Commit:** `c15d6a6`
**Applied fix:** Added `"crypto/subtle"` import. Replaced `if providedToken != storedToken` with `if subtle.ConstantTimeCompare([]byte(providedToken), []byte(storedToken)) != 1` to ensure constant-time comparison independent of token content, preventing byte-by-byte brute-force via response timing.

---

### WR-01: JWKS fetch does not check HTTP response status code

**Files modified:** `internal/auth/google/jwks.go`
**Commit:** `c5c345d` (combined with CR-02)
**Applied fix:** Added `if resp.StatusCode != http.StatusOK` check in `fetchLocked` immediately after the HTTP response is received and before any JSON decoding. Non-200 responses return a formatted error with the status code, preventing an error body from silently overwriting the key cache.

---

### WR-02: `Logout` handler silently discards Redis errors

**Files modified:** `internal/auth/handler/auth_handler.go`
**Commit:** `1f85792` (combined with CR-01)
**Applied fix:** Replaced `_ = h.svc.Logout(ctx, userID)` with an `if err != nil` block that calls `logger.FromCtx(ctx).Warn().Err(err).Str("userId", ...).Msg("logout redis error")`. The handler still returns HTTP 204 to the client for best-effort logout behaviour per D-06, but operational failures are now visible in logs.

---

### WR-03: All new Google users get the hardcoded username `"Player"`

**Files modified:** `internal/auth/repository/user_repo.go`, `internal/auth/service/auth_service.go`
**Commit:** `aa5ee79`
**Applied fix:** Extracted a new `GenerateUniqueUsername(ctx)` method from `CreateGuestUser`'s inline retry loop. `CreateGuestUser` now calls this helper. `CreateGoogleUser`'s signature was simplified to remove the `displayName` parameter (callers no longer supply a name); it now calls `GenerateUniqueUsername` internally. Updated `auth_service.go` to call `s.userRepo.CreateGoogleUser(ctx, googleSub, email)` without the `"Player"` literal.

---

### WR-04: `GetUserProviderByProvider` leaks `pgx.ErrNoRows` across layer boundary

**Files modified:** `internal/auth/repository/user_repo.go`, `internal/auth/service/auth_service.go`
**Commit:** `aa5ee79` (combined with WR-03)
**Applied fix:** Added `var ErrNotFound = errors.New("not found")` as a domain error in the repository package. Updated `GetUserProviderByProvider` to check `errors.Is(err, pgx.ErrNoRows)` and return `ErrNotFound` in that case, wrapping all other errors with context. Updated `auth_service.go` to check `errors.Is(err, repository.ErrNotFound)` and removed the now-unused `"github.com/jackc/pgx/v5"` import from the service package.

---

_Fixed: 2026-06-04T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
