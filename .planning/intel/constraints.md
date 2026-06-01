# constraints.md

Technical constraints extracted from SPEC sources. Applied precedence: ADR > SPEC > PRD > DOC.
No ADR documents exist in this ingest set; SPEC is the highest-precedence type present.

---

## CONSTRAINT-001: No ORM — sqlc Only

type: api-contract / implementation constraint
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC), J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

GORM and generic repository patterns are prohibited. All database queries must be
written with sqlc. Generated code is wrapped by the repository layer only.
No SELECT * — explicit column lists required in every query.

---

## CONSTRAINT-002: No Client-Submitted Game State

type: protocol / anti-cheat
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC)

Clients must never submit: score, combo, progress, rewards, rank changes, or
match results. Server rejects any such fields in incoming messages.
Solution grid must never be included in any server response.

---

## CONSTRAINT-003: Stateless Battle API (Load from Redis)

type: nfr / architecture
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC)

The Battle Engine API must be stateless. Battle state is always loaded from
Redis on every request. In-memory state is never relied upon for correctness.
This is required for horizontal scaling, reconnect, and pod-restart safety.

---

## CONSTRAINT-004: Handler Layer Cannot Call Repository Directly

type: architecture / layering
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC), J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC)

Handler -> Service -> Repository is the only valid call chain.
Handler -> Repository is explicitly prohibited.
Only the Service layer may initiate database transactions.
Repository must never create transactions.

---

## CONSTRAINT-005: Context-First Signatures on Public Methods

type: api-contract
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

Every public method must accept context.Context as its first parameter.
This applies across handler, service, and repository layers.

---

## CONSTRAINT-006: UUID v7 for All Primary Keys

type: schema
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

All primary keys in PostgreSQL tables use UUID v7. No auto-increment integers.
Applies to all tables: users, wallets, wallet_transactions, matches, match_players,
match_moves, missions, user_missions, shop_items, inventory_items.

---

## CONSTRAINT-007: UTC Timestamps Only

type: schema
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

All timestamps stored and returned as UTC (TIMESTAMPTZ in PostgreSQL, time.Time
in UTC in Go). Local timezone storage is prohibited.

---

## CONSTRAINT-008: Structured Logging Only — No fmt.Println

type: nfr / observability
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

Production code must use structured logging (e.g., zerolog). fmt.Println and
unstructured log output are prohibited in production paths.
All log entries must include relevant context fields (user_id, match_id, etc.).

---

## CONSTRAINT-009: WebSocket Envelope Format

type: protocol / api-contract
source: J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md (SPEC)

All WebSocket messages (both directions) must use the envelope:
{ "event": "<event-name>", "data": {} }
Malformed envelopes must be rejected. Both event and data fields are required.

---

## CONSTRAINT-010: Redis Key Naming Convention

type: schema / protocol
source: J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md (SPEC)

All Redis keys must follow the pattern: module:resource:id
Canonical key patterns:
- match:{matchId}:state — Hash, TTL 2 hours
- user:{userId}:connection — Hash, TTL 60 seconds
- queue:{region}:{difficulty}:{stake} — Sorted Set, score = join timestamp
- rate:user:{userId}:move — String (counter), TTL 1 second
- match:{matchId}:reconnect:{userId} — Hash, TTL 30 seconds

---

## CONSTRAINT-011: No Microservice Split at MVP

type: nfr / architecture
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC), J:\sources\sudoku-pvp\.planning\PROJECT.md

Microservice extraction is explicitly deferred post-MVP. The codebase is
structured as a modular monolith under internal/. The architecture must remain
compatible with future extraction but must not implement it at MVP.

---

## CONSTRAINT-012: Move Rate Limit — 5 Moves Per Second

type: nfr / anti-cheat
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC)

Server enforces a hard rate limit of 5 moves per second per user during battle.
Enforced via Redis counter with 1-second TTL. Violation escalation:
1st offense: warning, 2nd offense: temporary input lock, 3rd offense: match forfeit.
Client-side lock animation is cosmetic only; server-side lock is authoritative.

---

## CONSTRAINT-013: Match Finalization Belongs to Match Aggregate Only

type: api-contract / domain
source: J:\sources\sudoku-pvp\docs\DDD_AGGREGATE_RULES.md (DOC), J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC)

Only the Match aggregate may transition a match to FINISHED status and finalize
outcomes. No other module may directly write match result, coin_change, or rank_change
to match_players. Cross-aggregate effects (wallet, ranking) are triggered via
domain events on match.finished routing key.

---

## CONSTRAINT-014: Wallet Balance Non-Negative Invariant

type: schema / domain
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md (SPEC)

CoinBalance >= 0 and GemBalance >= 0 must be enforced as hard invariants in the
Wallet service before any persistence call. Negative balance is a rejected operation.

---

## CONSTRAINT-015: Puzzle Immutability Post-Creation

type: domain / schema
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md (SPEC)

SudokuPuzzle records are immutable after creation. puzzle_grid and solution_grid
may not be updated. A puzzle can be disabled (via admin) but not mutated.

---

## CONSTRAINT-016: No New Frameworks Without Approval

type: architecture
source: J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md (DOC)

No new third-party frameworks or libraries may be introduced without explicit
team approval. This constraint applies to all code generation contexts.

---

## CONSTRAINT-017: Code Generation Order

type: implementation process
source: J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md (DOC)

When generating backend code, the mandatory order is:
1. Database migrations
2. sqlc query files
3. Repository layer
4. Service layer
5. Handler layer
Unit tests, integration tests, interfaces, DTOs, and input validation must
accompany each layer.

---

## CONSTRAINT-018: SLO — Move Validation P95 < 20ms

type: nfr / performance
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC), J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD (PRD)

Battle engine move validation must complete in under 20ms at P95.
WebSocket broadcast must complete in under 100ms at P95.
Reconnect state recovery must complete in under 1 second.

---

## CONSTRAINT-019: Dead-Letter Retry — Max 3 Retries

type: protocol / messaging
source: J:\sources\sudoku-pvp\docs\RABBITMQ_TOPOLOGY.md (SPEC)

Failed messages on all RabbitMQ queues are retried at most 3 times before
routing to game.dlx (dead-letter exchange) and game.dead.queue.
Consumers must be idempotent to handle retry delivery.

---

## CONSTRAINT-020: Minimum Test Coverage Gate

type: nfr / quality
source: J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md (DOC), J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

CI must enforce >= 80% unit test coverage for: battle engine, ranking service, wallet service.
Test pyramid ratios: 70% unit, 20% integration, 10% E2E.
Integration tests must use testcontainers-go.
