# Phase 2: Auth & Identity - Context

**Gathered:** 2026-06-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver guest login, Google login, JWT issue/refresh/revoke, and auth middleware for REST endpoints and the WebSocket handshake. No user profile reads/writes (Phase 3). No admin roles (Phase 7).

</domain>

<decisions>
## Implementation Decisions

### Refresh Token Storage
- **D-01:** Refresh tokens stored in **Redis** with TTL. Key pattern: `refresh:{userId}`. Single token per user — new login overwrites previous (no multi-session).
- **D-02:** Access token TTL = **15 minutes**. Refresh token TTL = **30 days** (matches Redis key TTL).
- **D-03:** **Rotate on every use** — on `POST /auth/refresh`, old token is deleted from Redis and a new token+key are issued. Detects replay attacks.
- **D-04:** Signing algorithm: **HS256**. Secret from env var `SUDOKU_AUTH_JWT_SECRET`.
- **D-05:** JWT access token claims: `sub` (userId UUID v7), `role` ("player" | "admin"), `iat`, `exp`. No username or guest flag embedded — avoids stale claim issues.
- **D-06:** **POST /auth/logout** implemented in this phase — deletes `refresh:{userId}` key from Redis. Client must discard its access token.
- **D-07:** No per-device sessions. One refresh token per user globally.

### Google OAuth Integration
- **D-08:** Flow = **client-side ID token verification**. Mobile client completes Google Sign-In SDK flow, obtains a Google ID token (JWT), sends it to `POST /auth/google`. Backend verifies the token signature.
- **D-09:** Google JWKS cached **in memory** with ~1hr TTL. Token verified locally using cached public keys. On `kid` mismatch, re-fetch JWKS immediately. No per-login call to Google tokeninfo endpoint.
- **D-10:** Google identity stored in a **separate `user_providers` table**: columns `(user_id, provider, provider_user_id, email, created_at)`. `provider` = "google". No `google_id` column on `users` table.
- **D-11:** Email stored in `user_providers` only — NOT added to `users` table.
- **D-12:** First Google login **auto-creates** a new `users` row + `user_providers` row + `wallets` row in a single DB transaction. Subsequent logins look up by `(provider, provider_user_id)`.
- **D-13:** New migration required: `000006_create_user_providers.up.sql`.

### Guest Identity
- **D-14:** `POST /auth/guest` creates a `users` row instantly with no `user_providers` entry. Identity = userId UUID only.
- **D-15:** Guest username format: **Adjective + Noun + number** (e.g. `SwiftKing42`). Embedded word lists (no external library). Collision retry: regenerate if username already taken.
- **D-16:** Guest accounts cannot be upgraded or linked to a Google account. Google login always creates a distinct new account. No merge path.
- **D-17:** Guests have role = "player" and full access to all player endpoints. No feature restrictions in this phase.

### WebSocket Auth Mechanism
- **D-18:** `GET /ws/connect` authenticates via **Authorization header** (`Bearer {accessToken}`). Confirmed native mobile-only clients — no browser WebSocket clients anticipated.
- **D-19:** Auth failure before upgrade = **HTTP 401**. Connection is never established for an invalid/expired token.
- **D-20:** Shared JWT validation logic: one `ValidateJWT(token string) (*Claims, error)` function in `internal/middleware`. Both REST middleware and WS middleware call it — extraction from Authorization header is the same pattern for both transports.
- **D-21:** After successful validation, middleware sets `c.Set("userId", claims.Sub)` and `c.Set("role", claims.Role)` in Gin context. Handlers read these values — no re-parsing.

### Apple Login
- **D-22:** Apple login (`POST /auth/apple`) is **deferred** — not in Phase 2 scope. Product owner confirmation required before any phase includes it. `REQ-auth-apple` remains flagged in STATE.md.

### Claude's Discretion
- JWKS cache implementation detail (sync.RWMutex vs sync.Map, exact cache struct shape) — follow existing Go patterns in the codebase.
- Error response body format for auth failures — follow existing Gin error response conventions once established.
- Word list source for guest username generation — embed a small curated list; no dictionary library required.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API Contract
- `docs/API_CONTRACT.md` — Auth endpoints: `POST /auth/guest`, `POST /auth/google`, `POST /auth/refresh`. WebSocket: `GET /ws/connect`.

### Requirements
- `docs/PRODUCT_REQUIREMENTS.MD` — Auth section: guest login, Google login, JWT flows.
- `docs/BACKEND_ARCHITECTURE.md` — Auth Module section (endpoints, responsibilities); Architecture Philosophy.

### Database Schema
- `docs/DATABASE_SCHEMA.md` — `users` table definition. New `user_providers` table must follow same conventions (UUID PK, UTC timestamps).

### Redis Schema
- `docs/REDIS_SCHEMA.md` — Authoritative Redis key conventions. New key `refresh:{userId}` must be documented here (or confirmed with researcher).

### Planning Artifacts
- `.planning/REQUIREMENTS.md` — REQ-auth-guest, REQ-auth-google, REQ-auth-refresh, REQ-auth-jwt-middleware (auth section).
- `.planning/PROJECT.md` — DEC-014 (layering rules), DEC-013 (aggregate ownership).

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/middleware/logger.go` — Pattern for Gin middleware using `c.Request.Context()` (not `c.Copy()`). Auth middleware must follow this same context-propagation pattern to preserve OTel trace IDs.
- `internal/middleware/tracing.go` — OTel tracing middleware already registered. Auth middleware must be registered after tracing so trace context is available in auth logs.
- `internal/config/config.go` — `Config` struct needs a new `AuthConfig` sub-struct: `JWTSecret string`, `AccessTokenTTL time.Duration`, `RefreshTokenTTL time.Duration`.

### Established Patterns
- Handler → Service → Repository layering (DEC-014). Auth handlers must be thin; JWT logic in auth service; token persistence in auth repository (Redis).
- Context-first public method signatures: `func (s *AuthService) Login(ctx context.Context, ...) (...)`.
- UUID v7 for all PKs. UTC timestamps. `sqlc` only — no `SELECT *`.
- Structured logging via `logger.FromCtx(ctx)` — no `fmt.Println`.

### Integration Points
- `db/migrations/` — New migration `000006_create_user_providers.up.sql` needed before sqlc query generation.
- `internal/redis/` — Auth repository uses existing Redis client for refresh token storage.
- `internal/app/` — Auth routes registered on Gin router; auth middleware applied to protected route groups.
- `internal/middleware/` — JWT middleware function lives here alongside existing logger/tracing middleware.

</code_context>

<specifics>
## Specific Ideas

- Guest username: `Adjective + Noun + number` format (e.g. `SwiftKing42`). Embedded word list, simple collision retry.
- JWKS caching: in-memory struct with `sync.RWMutex`, re-fetch triggered by `kid` not found in cache.
- Refresh token key: `refresh:{userId}` — single string value (the opaque token or hashed token), TTL = 30 days.

</specifics>

<deferred>
## Deferred Ideas

- **Apple login** (`POST /auth/apple`) — needs product owner confirmation. `REQ-auth-apple` flagged. Defer to a future phase once scope is confirmed.
- **Per-device session management** — multiple refresh tokens per user, device-ID keyed. Out of scope for MVP.
- **Guest restrictions on ranked PvP** — role-based feature gating for guests belongs in Phase 5 matchmaking.
- **Guest-to-Google account linking/merge** — no upgrade path in MVP; would require a separate `POST /auth/link/google` flow.

</deferred>

---

*Phase: 2-Auth & Identity*
*Context gathered: 2026-06-04*
