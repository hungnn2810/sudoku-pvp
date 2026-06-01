# PROJECT.md

## Project
- Name: Sudoku PvP Backend
- Type: Greenfield backend service
- Primary stack: Go, Gin, PostgreSQL, Redis, RabbitMQ, WebSocket

## Vision
Build a server-authoritative, deterministic backend for real-time Sudoku PvP with ranking, wallet economy, missions, shop, and operations tooling.

## Goals
- Deliver deterministic and idempotent API and battle behavior.
- Support real-time PvP with reconnect safety and horizontal scaling.
- Maintain secure economy and ranking progression.
- Provide operational observability and admin controls.

## Non-Goals (MVP)
- Microservice split at initial launch.
- Client-authoritative scoring or match resolution.
- Unbounded feature expansion outside defined docs scope.

## Locked Decisions
- Architecture: Modular monolith with domain-oriented modules.
- API versioning: `/api/v1`.
- Realtime transport: WebSocket endpoint `/ws/connect`.
- Persistence: PostgreSQL (source of truth), Redis (ephemeral/realtime state), RabbitMQ (domain/event fanout).
- Battle engine: server-authoritative and deterministic.
- Coding standards: handler -> service -> repository layering; context-first signatures; structured logging; no `SELECT *`.

## Extended Decisions (from ingest)

### DEC-006: Battle State Persistence Model
Realtime battle state in Redis (match:{matchId}:state, TTL 2h). Historical data to PostgreSQL after match completion. Enables horizontal scaling, reconnect support, pod restart recovery.

### DEC-007: Scoring and Combo Rules
Correct +10, Wrong -5+reset+lock(2s), Row +20, Column +20, Box +15, Puzzle +100. Combo: 1.0x/1.2x/1.4x. Wrong move resets combo to 0.

### DEC-008: Match Lifecycle State Machine
CREATED -> WAITING_PLAYERS -> READY -> COUNTDOWN(3s) -> PLAYING -> FINAL_COUNTDOWN(30s) -> FINISHED. Durations: Easy=300s, Medium=420s, Hard=600s, Expert=720s.

### DEC-009: Winner Determination
Rule 1: higher score wins. Rule 2: earlier finish time. Rule 3: draw. On timeout: highest score wins. Surrender: immediate loss with rank and coin loss.

### DEC-010: Disconnect and Reconnect Policy
Grace period: 30 seconds. Match continues during disconnect. On reconnect: battle.reconnect_state sent (score, combo, progress, remainingSeconds). Redis key: match:{matchId}:reconnect:{userId}, TTL 30s. After 30s: auto-surrender.

### DEC-011: Anti-Cheat Rate Limiting
5 moves/sec limit. Redis key: rate:user:{userId}:move, TTL 1s. Violation escalation: warning -> input lock -> match forfeit.

### DEC-012: RabbitMQ Event Topology
Exchange: game.events (topic). Queues: ranking.queue, wallet.queue, mission.queue, analytics.queue, notification.queue. Routing keys: match.started, match.finished, wallet.transaction.created, ranking.changed, mission.completed, shop.purchased. DLX: game.dlx with game.dead.queue, 3 retries.

### DEC-013: Aggregate Ownership
Wallet exclusively owns balance mutations (CoinBalance >= 0, GemBalance >= 0). Match exclusively finalizes outcomes. Cross-aggregate coordination via domain events only — no direct aggregate mutation.

### DEC-014: Code Generation and Layering
Handler -> Service -> Repository (no Handler->Repository direct calls). Only Service may initiate DB transactions. All public methods: context.Context first param. sqlc only — no SELECT *, explicit columns. PKs: UUID v7. Timestamps: UTC. Generation order: migrations -> sqlc -> repositories -> services -> handlers.

### DEC-015: Redis Queue Key Schema
Queue key: queue:{region}:{difficulty}:{stake} (Sorted Set, score = join timestamp). REDIS_SCHEMA.md authoritative; BACKEND_ARCHITECTURE.md shorthand was superseded.

### DEC-016: Admin Roles
Four roles: SuperAdmin, GameAdmin, SupportAdmin, ReadOnlyAdmin. Audit log covers: admin login, user changes, economy changes, match actions. All immutable append-only records.

## Source Documents
- `docs/PRODUCT_REQUIREMENTS.MD`
- `docs/BACKEND_ARCHITECTURE.md`
- `docs/API_CONTRACT.md`
- `docs/BATTLE_ENGINE_SPEC.md`
- `docs/WEBSOCKET_PROTOCOL.md`
- `docs/DATABASE_SCHEMA.md`
- `docs/REDIS_SCHEMA.md`
- `docs/RABBITMQ_TOPOLOGY.md`
- `docs/DOMAIN_MODEL.md`
- `docs/DDD_AGGREGATE_RULES.md`
- `docs/EVENT_CATALOG.md`
- `docs/TESTING_STRATEGY.md`
- `docs/ENGINE_TASKS.md`
- `docs/ADMIN_PANEL_SPEC.md`
