# Phase 3: User & Wallet Core - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-04
**Phase:** 3-User & Wallet Core
**Areas discussed:** Module layout, PATCH /me field scope, Wallet commands in phase 3, Pagination strategy

---

## Module Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Separate modules | internal/user/ + internal/wallet/ as parallel top-level modules, each with handler/service/repository. Mirrors internal/auth/. | ✓ |
| Single user module | internal/user/ containing both profile and wallet sub-packages. Simpler imports but blurs DDD boundary. | |

**User's choice:** Separate modules — `internal/user/` and `internal/wallet/`
**Notes:** User confirmed the DDD boundary matters — wallet aggregate is independent of user profile aggregate.

---

## PATCH /me Field Scope

| Option | Description | Selected |
|--------|-------------|----------|
| username + avatarUrl | Both mutable. Username requires uniqueness re-check (3-50 chars). avatarUrl stored as-is. System fields not writable. | ✓ |
| avatarUrl only | Username fixed after creation. No uniqueness re-check needed. | |

**User's choice:** username + avatarUrl
**Notes:** Username update requires uniqueness check (409 on conflict). System fields (level, exp, rank_tier, rank_point) are not writable via PATCH /me.

---

## Wallet Commands in Phase 3

| Option | Description | Selected |
|--------|-------------|----------|
| Full command layer | CreditWallet + DebitWallet service methods with invariants. No HTTP endpoints. Later phases import wallet.Service. | ✓ |
| Read-only this phase | Only GET /wallet and GET /wallet/transactions. Mutations deferred to first phase that needs them. | |

**User's choice:** Full command layer
**Notes:** CreditWallet and DebitWallet are internal service methods only — no player-facing HTTP endpoints expose them in phase 3. DebitWallet enforces non-negative invariant atomically (balance check + ledger write + balance update in one DB transaction).

---

## Pagination Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Offset-based | ?page=1&limit=20. Simple. Acceptable for append-only data. Default 20, max 100. | ✓ |
| Cursor-based | ?cursor=<token>&limit=20. Stable under concurrent inserts. More complex to implement. | |

**User's choice:** Offset-based
**Notes:** Response envelope: `{ "items": [...], "total": int, "page": int, "limit": int }`. Applies to both GET /wallet/transactions and GET /matches.

---

## Claude's Discretion

- `PaginationParams` struct placement (`internal/common/` vs per-module) — follow existing common patterns
- Error response format for wallet invariant violations — follow `internal/common/response.go`
- SQL query file organization under `db/queries/` — follow auth.sql pattern

## Deferred Ideas

- HTTP wallet mutation endpoints (POST /wallet/credit etc.) — no player-facing mutation endpoints in MVP
- Cursor-based pagination — deferred; offset sufficient for MVP
- Match detail rich response (replay/moves) — deferred to Phase 4/5
