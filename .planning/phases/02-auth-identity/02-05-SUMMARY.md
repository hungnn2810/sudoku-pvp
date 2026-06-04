---
phase: 02-auth-identity
plan: "05"
subsystem: auth, handler, middleware, app
tags: [auth, gin, handler, middleware, jwt, websocket, routes]
dependency_graph:
  requires:
    - 02-04 (service.AuthService, service.TokenPair, service.(*AuthService).GuestLogin|GoogleLogin|Refresh|Logout)
    - 02-02 (jwt.ValidateJWT, jwt.Claims)
  provides:
    - handler.AuthHandler
    - handler.NewAuthHandler
    - handler.(*AuthHandler).GuestLogin
    - handler.(*AuthHandler).GoogleLogin
    - handler.(*AuthHandler).Refresh
    - handler.(*AuthHandler).Logout
    - middleware.JWTMiddleware
    - middleware.WSJWTMiddleware
    - common.Error
    - common.OK
  affects:
    - internal/app/app.go (auth wired; 5 routes registered)
tech_stack:
  added: []
  patterns:
    - thin Gin handler delegating to AuthService; no business logic in handlers
    - Bearer token extraction via extractBearerToken helper shared by JWTMiddleware and WSJWTMiddleware
    - c.Abort() on auth failure prevents downstream handler execution (T-02-05-01, T-02-05-03)
    - userId set as string UUID in Gin context from validated JWT claims (D-21)
    - JWTMiddleware on route groups (not global) preserving /health and /metrics (T-02-05-05)
    - wsPlaceholderHandler returns 503 until Phase 5 WebSocket implementation
key_files:
  created:
    - internal/auth/handler/auth_handler.go
    - internal/middleware/auth.go
  modified:
    - internal/app/app.go
    - internal/common/response.go
decisions:
  - "common.Error and common.OK helpers implemented in response.go (was a TODO stub)"
  - "userId stored as claims.Sub.String() in Gin context per D-21 must_haves spec"
  - "extractBearerToken private helper shared between JWTMiddleware and WSJWTMiddleware (D-20)"
  - "jwtSecret converted to []byte once in setupRouter and reused for both middleware calls"
  - "auth wired in New() step 4.5 — after Redis+Postgres, before RabbitMQ and router setup"
metrics:
  duration: "~15 minutes"
  completed_date: "2026-06-04"
  tasks_completed: 3
  tasks_total: 3
  files_created: 2
  files_modified: 2
---

# Phase 2 Plan 05: Handlers, Middleware, and Routes Summary

## One-liner

Thin Gin handlers for four auth endpoints, JWT/WS middleware sharing ValidateJWT, and full auth dependency wiring in app.go with five routes registered across public and protected route groups.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Implement internal/auth/handler/auth_handler.go | 3ca0b3f | internal/auth/handler/auth_handler.go, internal/common/response.go |
| 2 | Implement internal/middleware/auth.go (REST + WS JWT middleware) | 00bce2e | internal/middleware/auth.go |
| 3 | Wire auth in internal/app/app.go and register routes | ad030ad | internal/app/app.go |

## Artifacts Produced

### New Types
- `handler.AuthHandler` struct — field: svc *service.AuthService

### New Functions
- `handler.NewAuthHandler(svc *service.AuthService) *AuthHandler`
- `handler.(*AuthHandler).GuestLogin(c *gin.Context)` — delegates to svc.GuestLogin
- `handler.(*AuthHandler).GoogleLogin(c *gin.Context)` — parses idToken, delegates to svc.GoogleLogin
- `handler.(*AuthHandler).Refresh(c *gin.Context)` — parses userId+refreshToken, delegates to svc.Refresh
- `handler.(*AuthHandler).Logout(c *gin.Context)` — reads userId from Gin context, delegates to svc.Logout
- `middleware.JWTMiddleware(secret []byte) gin.HandlerFunc` — REST Bearer token validation
- `middleware.WSJWTMiddleware(secret []byte) gin.HandlerFunc` — WebSocket handshake token validation
- `common.Error(c *gin.Context, status int, code string, msg string)` — standard JSON error response helper
- `common.OK(c *gin.Context, data any)` — standard JSON 200 response helper
- `app.wsPlaceholderHandler(c *gin.Context)` — 503 placeholder until Phase 5 WS implementation

### Private Helpers
- `middleware.extractBearerToken(c *gin.Context) string` — shared Bearer extraction for both middleware

### Modified Functions / Types
- `app.App` struct — added `authHandler *handler.AuthHandler` field
- `app.New()` — step 4.5: wires JWKSCache, AuthRepo, UserRepo, AuthService, AuthHandler
- `app.setupRouter()` — registers 5 auth routes; applies JWTMiddleware to protected group

### Routes Registered
- `POST /api/v1/auth/guest` — public; creates guest user
- `POST /api/v1/auth/google` — public; Google ID token verification
- `POST /api/v1/auth/refresh` — public; token rotation
- `POST /api/v1/auth/logout` — protected (JWTMiddleware); clears refresh token
- `GET /ws/connect` — WSJWTMiddleware + placeholder; returns 503 until Phase 5

## Verification Results

All plan verification checks passed:
- `go build ./...` exits 0
- `go vet ./...` exits 0
- `grep "c.Abort()" internal/middleware/auth.go` — 4 actual abort calls (2 per middleware)
- POST routes for /auth/guest, /google, /refresh, /logout all registered in app.go
- GET /ws/connect registered with WSJWTMiddleware
- `grep "authjwt.ValidateJWT" internal/middleware/auth.go` — shared validation confirmed (D-20)
- `grep -c "c.Request.Context()" internal/auth/handler/auth_handler.go` — 4 (one per handler)
- `grep "c.Copy()" internal/auth/handler/auth_handler.go` — empty (correct)
- `grep "logger.FromCtx" internal/auth/handler/auth_handler.go` — structured logging confirmed
- `grep "http.StatusNoContent" internal/auth/handler/auth_handler.go` — 204 on logout confirmed

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Implement common.Error and common.OK helpers**

- **Found during:** Task 1
- **Issue:** `internal/common/response.go` contained only an `ErrorResponse` struct and a TODO comment about adding gin-based helpers. The task instruction said "use common.Error and common.OK in every handler" but those functions did not exist yet.
- **Fix:** Replaced the TODO comment with working implementations of `common.Error` and `common.OK` using the correct `*gin.Context` pattern. This unblocked the handler compilation.
- **Files modified:** internal/common/response.go
- **Commit:** 3ca0b3f
- **Classification:** Rule 2 (missing critical functionality required for handler correctness).

## Threat Model Compliance

| Threat ID | Status | Notes |
|-----------|--------|-------|
| T-02-05-01 | Mitigated | c.Abort() before c.Next() in JWTMiddleware; downstream handlers never execute on auth failure |
| T-02-05-02 | Mitigated | Logout reads userId from Gin context set by JWTMiddleware; handler unreachable without valid token; client body userId not trusted |
| T-02-05-03 | Mitigated | WSJWTMiddleware registered before wsPlaceholderHandler; c.Abort() on failure prevents placeholder execution |
| T-02-05-04 | Accepted | idToken transmitted over TLS; short-lived (1hr); not logged in handler |
| T-02-05-05 | Mitigated | JWTMiddleware applied to specific route groups after global chain; /health and /metrics unprotected as intended |

## Known Stubs

| Stub | File | Line | Reason |
|------|------|------|--------|
| wsPlaceholderHandler returns 503 | internal/app/app.go | ~168 | Intentional per plan — WebSocket handler implemented in Phase 5; WSJWTMiddleware is functional; only the upgrade handler is stubbed |

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new_endpoints | internal/app/app.go | 5 new HTTP endpoints introduced: POST /auth/guest, /google, /refresh, /logout, GET /ws/connect — all covered by plan threat model T-02-05-01 through T-02-05-05 |

## Self-Check: PASSED
