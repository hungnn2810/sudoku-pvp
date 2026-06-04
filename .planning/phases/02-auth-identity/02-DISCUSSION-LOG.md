# Phase 2: Auth & Identity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-04
**Phase:** 2-Auth & Identity
**Areas discussed:** Refresh token storage, Google OAuth pattern, Guest identity & upgrade, WebSocket auth mechanism

---

## Refresh Token Storage

| Option | Description | Selected |
|--------|-------------|----------|
| Redis with TTL | Token hash stored in Redis with TTL matching refresh token expiry. Key: refresh:{userId} | ✓ |
| DB refresh_tokens table | Durable, survives Redis flush, supports listing active sessions per user | |
| Stateless (no storage) | Refresh token is just a long-lived JWT — no revocation possible | |

**User's choice:** Redis with TTL

---

| Option | Description | Selected |
|--------|-------------|----------|
| Access 15min / Refresh 30 days | Standard mobile game TTLs | ✓ |
| Access 1hr / Refresh 7 days | Longer access window, shorter refresh | |
| Access 15min / Refresh 90 days | Very long refresh, higher risk | |

**User's choice:** Access 15min / Refresh 30 days

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, rotate on every use | Old token invalidated on use, new one issued. Detects token theft | ✓ |
| No rotation | Same refresh token reused until expiry | |

**User's choice:** Yes, rotate on every use

---

| Option | Description | Selected |
|--------|-------------|----------|
| HS256 | Symmetric HMAC-SHA256, single secret, fast | ✓ |
| RS256 | Asymmetric RSA, useful for multi-service verification | |

**User's choice:** HS256

---

| Option | Description | Selected |
|--------|-------------|----------|
| sub + role only | Minimal claims; no stale data risk | ✓ |
| sub + role + username + guest flag | Embeds username; stale if updated | |
| sub only | Every request hits DB for role/username | |

**User's choice:** sub + role only

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — POST /auth/logout | Deletes refresh token from Redis | ✓ |
| No — defer to later phase | Let tokens expire naturally | |

**User's choice:** Yes — POST /auth/logout (in this phase)

---

| Option | Description | Selected |
|--------|-------------|----------|
| No — single token per user | One refresh token per user globally | ✓ |
| Yes — per-device tokens | Key: refresh:{userId}:{deviceId}, multiple active sessions | |

**User's choice:** No — single token per user

---

## Google OAuth Pattern

| Option | Description | Selected |
|--------|-------------|----------|
| Client sends ID token | Mobile client gets Google ID token, sends to backend for verification | ✓ |
| Server-side OAuth code exchange | Backend exchanges authorization code with Google token endpoint | |

**User's choice:** Client sends ID token

---

| Option | Description | Selected |
|--------|-------------|----------|
| Add google_id column to users table | Simple, nullable column | |
| Separate user_providers table | user_id, provider, provider_user_id, email — supports multiple providers | ✓ |

**User's choice:** Separate user_providers table
**Notes:** User chose extensibility over simplicity — enables Apple login and future providers without schema changes.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Email in user_providers only | Identity managed through providers, users table stays email-free | ✓ |
| Email in both tables | Useful for admin lookup but requires sync across providers | |

**User's choice:** Email in user_providers only

---

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-create user on first Google login | Create users + user_providers + wallet in one transaction | ✓ |
| Require prior registration | Return 404 if unknown, separate /auth/register needed | |

**User's choice:** Auto-create on first Google login

---

| Option | Description | Selected |
|--------|-------------|----------|
| Cache JWKS with TTL | Fetch once, verify locally, re-fetch on kid mismatch | ✓ |
| Call tokeninfo on every login | Simple but 50-200ms per-login latency, external dependency | |

**User's choice:** Cache JWKS with TTL (~1hr in-memory)

---

## Guest Identity & Upgrade

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-generated username + users row only | No user_providers entry, identity = userId only | ✓ |
| Device ID as identifier | Maps device fingerprint → user_id for reconnect across reinstall | |

**User's choice:** Auto-generated username + users row only

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — link via POST /auth/link/google | Guest links Google account, keeps all existing data | |
| No — Google login always creates new account | Guest and Google accounts completely separate | ✓ |

**User's choice:** No — Google login always creates a new account
**Notes:** Guest progress is not preserved when player signs in with Google. Simpler implementation chosen.

---

| Option | Description | Selected |
|--------|-------------|----------|
| No restrictions — full access | Same JWT, same capabilities as Google-linked users | ✓ |
| Restrict guests from ranked PvP | Role check in matchmaking — out of Phase 2 scope | |

**User's choice:** No restrictions — full access in this phase

---

| Option | Description | Selected |
|--------|-------------|----------|
| Player_ + 6 hex chars | e.g. Player_a3f9b2, derived from UUID bytes | |
| Adjective + Noun + number | e.g. SwiftKing42, more human-readable and memorable | ✓ |

**User's choice:** Adjective + Noun + number
**Notes:** Embedded word list; retry on username collision.

---

## WebSocket Auth Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| JWT as query param ?token=... | Simple, works in all clients; token visible in server logs | |
| First message auth frame | Token never in URL; stateful handshake complexity | |
| Authorization header | Standard HTTP; works in native mobile clients, NOT browsers | ✓ |

**User's choice:** Authorization header

---

| Option | Description | Selected |
|--------|-------------|----------|
| No — native mobile only | Confirmed, no browser clients expected | ✓ |
| Web client possible in future | Would require rework of WS auth mechanism | |

**User's choice:** No — native mobile only (confirmed)

---

| Option | Description | Selected |
|--------|-------------|----------|
| Reject upgrade with HTTP 401 | Clean, client knows to refresh token and reconnect | ✓ |
| Upgrade then send error frame | Connection established briefly before auth confirmed | |

**User's choice:** HTTP 401 before upgrade

---

| Option | Description | Selected |
|--------|-------------|----------|
| Shared middleware, different extraction | One ValidateJWT function; both REST and WS call it | ✓ |
| Separate middleware per transport | Independent but duplicated logic | |

**User's choice:** Shared middleware

---

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — store in Gin context | c.Set("userId", ...) and c.Set("role", ...) after validation | ✓ |
| Parse token again in each handler | Wasteful re-parsing, no shared state | |

**User's choice:** Store in Gin context

---

## Claude's Discretion

- JWKS cache implementation details (sync.RWMutex vs sync.Map, cache struct shape)
- Error response body format for auth failures
- Word list source for guest username generation

## Deferred Ideas

- Apple login — needs product owner confirmation before scoping into any phase
- Per-device session management — multiple refresh tokens keyed by device ID
- Guest restrictions on ranked PvP — belongs in Phase 5
- Guest-to-Google account linking/merge — no upgrade path in MVP
