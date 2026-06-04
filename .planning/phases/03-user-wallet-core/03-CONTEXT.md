# Phase 3: User & Wallet Core - Context

**Gathered:** 2026-06-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver user profile read/update APIs, wallet balance read, wallet transaction history read, and wallet mutation commands (CreditWallet/DebitWallet) with full invariants. No HTTP mutation endpoints for wallet in this phase — mutations are internal service methods consumed by later phases (5, 6). No match data implementation (Phase 4). No ranking, missions, or shop (Phase 6).

</domain>

<decisions>
## Implementation Decisions

### Module Layout
- **D-01:** `internal/user/` and `internal/wallet/` as independent top-level modules, each with `handler/`, `service/`, `repository/` sub-packages. Parallel to `internal/auth/`. Clean DDD boundary — wallet aggregate is independent of user profile aggregate.
- **D-02:** Directory structure:
  ```
  internal/
    user/
      handler/
      service/
      repository/
    wallet/
      handler/
      service/
      repository/
  ```

### PATCH /me Field Scope
- **D-03:** PATCH /me accepts two optional fields: `username` (3-50 chars, uniqueness re-check required — same logic as registration) and `avatarUrl` (any non-empty string, no format validation). Partial update — omitted fields unchanged.
- **D-04:** System-managed fields (`level`, `exp`, `rank_tier`, `rank_point`) are NOT writable via PATCH /me. Any attempt to set them is ignored (or rejected with 400).
- **D-05:** Username update triggers `ExistsUsername` check before persisting. If taken, return 409 Conflict.

### Wallet Mutation Commands
- **D-06:** Phase 3 implements `CreditWallet(ctx, userID, amount, txType, refType, refID)` and `DebitWallet(ctx, userID, amount, txType, refType, refID)` in `wallet/service`. These are internal service methods — no HTTP endpoints expose them in phase 3.
- **D-07:** `DebitWallet` enforces non-negative invariant: fetch current `coin_balance` → reject if `balance - amount < 0` → write `wallet_transactions` row → update `wallets.coin_balance`. All in a single DB transaction.
- **D-08:** `CreditWallet` follows same atomic pattern: write transaction row + update balance in one DB transaction.
- **D-09:** `wallet_transactions` is immutable — no UPDATE or DELETE queries. `balance_before` and `balance_after` stored on every record (REQ-wallet-immutable-ledger).
- **D-10:** `DebitWallet` returns a domain error (e.g., `ErrInsufficientBalance`) when balance would go negative. Callers (Phase 5, 6) handle this error.

### Pagination Strategy
- **D-11:** Offset-based pagination for GET /wallet/transactions and GET /matches. Query params: `?page=1&limit=20`. Default `limit=20`, max `limit=100`. Response envelope:
  ```json
  { "items": [...], "total": 142, "page": 1, "limit": 20 }
  ```
- **D-12:** Both endpoints apply identical pagination struct — define a shared `PaginationParams` in `internal/common/` or per-module; researcher to recommend.

### User Stats and Match History
- **D-13:** GET /me/stats queries `match_players` table for win/loss/draw counts. In phase 3, table is empty — returns zeros. No stubbing needed: live query against real schema.
- **D-14:** GET /matches queries `matches` + `match_players` tables with join. Returns paginated results. In phase 3, returns empty list. Match detail endpoint (GET /matches/:id) included.

### Claude's Discretion
- Exact `PaginationParams` struct placement (`internal/common/` vs per-module) — follow existing common patterns.
- Error response format for wallet invariant violations — follow `internal/common/response.go` conventions.
- SQL query file organization under `db/queries/` (user.sql, wallet.sql) — follow auth.sql pattern.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### API Contract
- `docs/API_CONTRACT.md` — User endpoints: GET /me, PATCH /me, GET /me/stats, GET /matches, GET /matches/:id. Wallet endpoints: GET /wallet, GET /wallet/transactions.

### Requirements
- `docs/PRODUCT_REQUIREMENTS.MD` — User profile and wallet sections.
- `docs/BACKEND_ARCHITECTURE.md` — User Module and Wallet Module sections; DDD aggregate rules.
- `docs/DOMAIN_MODEL.md` — User and Wallet aggregate definitions.
- `docs/DDD_AGGREGATE_RULES.md` — Aggregate ownership rules (DEC-013: Wallet exclusively owns balance mutations).

### Database Schema
- `docs/DATABASE_SCHEMA.md` — `users`, `wallets`, `wallet_transactions`, `match_players`, `matches` table definitions.
- `db/migrations/000001_create_users.up.sql` — users schema.
- `db/migrations/000002_create_wallets.up.sql` — wallets schema (user_id PK, coin_balance, gem_balance).
- `db/migrations/000003_create_wallet_transactions.up.sql` — wallet_transactions schema (immutable ledger, balance_before, balance_after).

### Redis Schema
- `docs/REDIS_SCHEMA.md` — Authoritative Redis key conventions (no new Redis keys expected in this phase, but confirm).

### Planning Artifacts
- `.planning/REQUIREMENTS.md` — REQ-user-profile-read, REQ-user-profile-update, REQ-user-stats, REQ-user-match-history, REQ-wallet-read, REQ-wallet-transactions, REQ-wallet-immutable-ledger, REQ-wallet-non-negative, REQ-wallet-aggregate-ownership.
- `.planning/PROJECT.md` — DEC-013 (aggregate ownership), DEC-014 (layering rules), DEC-015 (Redis queue key schema).

### Existing Code
- `internal/auth/repository/user_repo.go` — Contains `GetUserByID` query and `CreateWallet` — user repo pattern to follow; wallet repo must NOT duplicate wallet creation logic.
- `db/queries/auth.sql` — sqlc query pattern to replicate for user.sql and wallet.sql.
- `internal/common/errors.go` — ErrNotFound sentinel; new ErrInsufficientBalance sentinel follows same pattern.
- `internal/common/response.go` — Response envelope conventions.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/auth/repository/user_repo.go` — `GetUserByID` already implemented (sqlc query). User repository in this phase can reuse or wrap this query; avoid duplicating it.
- `internal/common/errors.go` — `ErrNotFound` sentinel. Add `ErrInsufficientBalance` here for wallet domain.
- `internal/common/response.go` — Gin response helpers. All new handlers use these.
- `internal/middleware/auth.go` — JWT middleware sets `c.Set("userId", ...)`. Handlers in user/wallet modules extract userId from Gin context — same pattern as auth phase.

### Established Patterns
- Handler → Service → Repository (DEC-014). No Handler → Repository direct calls.
- Only Service layer initiates DB transactions (pgx BeginTx + deferred Rollback + explicit Commit).
- Context-first public method signatures: `func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*ProfileDTO, error)`.
- sqlc only — all queries in `db/queries/*.sql`, no raw SQL strings in Go code.
- UUID v7 PKs, UTC timestamps, no `SELECT *`.
- Structured logging via `logger.FromCtx(ctx)`.

### Integration Points
- `db/queries/` — New files: `user.sql` (profile read/update, stats, match history) and `wallet.sql` (balance read, transaction list, credit/debit).
- `internal/app/app.go` — Register user and wallet route groups under `/api/v1/` with JWT middleware applied.
- `internal/user/repository/` — May call `GetUserByID` from existing auth sqlc queries, or define its own read queries. Researcher to decide duplication vs import.
- `internal/wallet/repository/` — Owns all wallet mutation queries. No other repository touches `wallets` or `wallet_transactions`.

</code_context>

<specifics>
## Specific Ideas

- `DebitWallet` signature: `(ctx context.Context, userID uuid.UUID, amount int64, txType, refType, refID string) error`
- `CreditWallet` signature: same pattern.
- Pagination response: `{ "items": [...], "total": int, "page": int, "limit": int }` — consistent across all paginated endpoints in phase 3 and future phases.
- Username update: reuse `ExistsUsername` sqlc query already defined in `db/queries/auth.sql`.
- New sqlc query needed: `UpdateUserProfile` (sets username, avatar_url, updated_at WHERE id = $1).

</specifics>

<deferred>
## Deferred Ideas

- **HTTP wallet mutation endpoints** (POST /wallet/credit, POST /wallet/debit) — no player-facing mutation endpoints needed in MVP. Admin economy management is Phase 7.
- **Cursor-based pagination** — deferred; offset-based sufficient for MVP append-only data.
- **Match detail rich response** — full match replay/moves data deferred to Phase 4/5 when match engine exists.

</deferred>

---

*Phase: 3-User & Wallet Core*
*Context gathered: 2026-06-04*
