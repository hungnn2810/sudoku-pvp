# decisions.md

Synthesized architectural decisions extracted from SPEC and DOC sources.
No ADR-type documents were present in this ingest set. Decisions below are
drawn from SPEC sources (BACKEND_ARCHITECTURE.md, BATTLE_ENGINE_SPEC.md,
DOMAIN_MODEL.md, and supporting specs) and the existing PROJECT.md locked
decisions block, which is treated as authoritative project-level record.

source: J:\sources\sudoku-pvp\.planning\PROJECT.md (existing locked decisions)
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC, high confidence)
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md (SPEC, high confidence)
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md (SPEC, high confidence)

---

## DEC-001: Modular Monolith Architecture

status: accepted (project-locked)
scope: overall system architecture
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\.planning\PROJECT.md

Do not build microservices initially. Use a modular monolith with domain-oriented
modules under internal/. The architecture must allow future extraction into
microservices. Microservice split is a non-goal for MVP.

---

## DEC-002: Primary Technology Stack

status: accepted (project-locked)
scope: language, framework, persistence, messaging, realtime
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\.planning\PROJECT.md

- Language: Go 1.25+
- HTTP Framework: Gin
- Database: PostgreSQL 17+ (source of truth)
- Query layer: sqlc (no GORM, no generic ORM)
- Cache / Ephemeral state: Redis 8+
- Messaging / Event fanout: RabbitMQ (topic exchange)
- Realtime transport: WebSocket (nhooyr/websocket recommended)
- Observability: OpenTelemetry + Prometheus + Loki + Grafana
- Deployment: Docker

---

## DEC-003: API Versioning and Base URL

status: accepted (project-locked)
scope: REST API routing
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\OPENAPI.yaml, J:\sources\sudoku-pvp\.planning\PROJECT.md

All REST endpoints are versioned under /api/v1. JWT Bearer authentication
is required for authenticated endpoints.

---

## DEC-004: WebSocket Connection Endpoint

status: accepted (project-locked)
scope: realtime transport
source: J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\.planning\PROJECT.md

Single WebSocket endpoint: GET /ws/connect
Authentication: Bearer JWT on handshake.
Envelope format: { "event": "<name>", "data": {} }
Heartbeat: client sends battle.ping every 15s; server responds battle.pong; timeout 45s.

---

## DEC-005: Server-Authoritative Battle Engine

status: accepted (project-locked)
scope: battle engine, scoring, results, anti-cheat
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\.planning\PROJECT.md

The battle engine is fully server-authoritative and deterministic.
Clients are never trusted for: score, combo, progress, rewards, rank changes,
match results, or move validation.
The solution grid is never transmitted to clients.
Same input must always produce the same output (no randomness during battle).

---

## DEC-006: Battle State Persistence Model

status: accepted
scope: battle engine state storage
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md

Realtime battle state stored in Redis (stateless API, load from Redis on every
request — enables horizontal scaling, reconnect support, pod restart recovery).
Historical data persisted to PostgreSQL after match completion.
Redis key: match:{matchId}:state, TTL: 2 hours.

---

## DEC-007: Scoring and Combo Rules

status: accepted
scope: battle engine scoring
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md

Correct move: +10
Wrong move: -5 and combo reset and 2-second input lock
Complete row: +20 (once per row)
Complete column: +20 (once per column)
Complete box (3x3): +15 (once per box)
Puzzle completion: +100
Combo multiplier: 1 correct = 1.0x, 2 consecutive = 1.2x, 3+ consecutive = 1.4x
Wrong move resets combo to 0.

---

## DEC-008: Match Lifecycle State Machine

status: accepted
scope: match and room lifecycle
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md

States in order: CREATED -> WAITING_PLAYERS -> READY -> COUNTDOWN (3s) ->
PLAYING -> FINAL_COUNTDOWN (30s) -> FINISHED
Final countdown triggered when first player completes puzzle.
Match durations: Easy=300s, Medium=420s, Hard=600s, Expert=720s.

---

## DEC-009: Winner Determination Rules

status: accepted
scope: match result calculation
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md

Rule 1: Higher score wins.
Rule 2: If scores equal, earlier finish time wins.
Rule 3: If still equal, draw.
On timeout (remaining time = 0): highest score wins.
Surrender: immediate loss with rank loss and coin loss.

---

## DEC-010: Disconnect and Reconnect Policy

status: accepted
scope: realtime reliability, reconnect handling
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md

Grace period: 30 seconds.
During disconnect: match continues, opponent continues playing.
On successful reconnect: server sends battle.reconnect_state with score, combo,
progress, remainingSeconds.
Reconnect state Redis key: match:{matchId}:reconnect:{userId}, TTL: 30s.
After 30s without reconnect: automatic surrender.

---

## DEC-011: Anti-Cheat Rate Limiting

status: accepted
scope: anti-cheat, move rate limiting
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md

Move rate limit: 5 moves per second.
Violation escalation: 1st = warning, 2nd = temporary input lock, 3rd = match forfeit.
Rate limit Redis key: rate:user:{userId}:move, TTL: 1 second.

---

## DEC-012: RabbitMQ Event Topology

status: accepted
scope: domain event messaging
source: J:\sources\sudoku-pvp\docs\RABBITMQ_TOPOLOGY.md, J:\sources\sudoku-pvp\docs\EVENT_CATALOG.md

Exchange: game.events (topic type)
Queues: ranking.queue, wallet.queue, mission.queue, analytics.queue, notification.queue
Routing keys: match.started, match.finished, wallet.transaction.created,
ranking.changed, mission.completed, shop.purchased
Retry strategy: 3 retries, dead letter exchange game.dlx, dead letter queue game.dead.queue

---

## DEC-013: Aggregate Ownership and Cross-Aggregate Rules

status: accepted
scope: domain model, DDD boundaries
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md, J:\sources\sudoku-pvp\docs\DDD_AGGREGATE_RULES.md

Wallet aggregate exclusively owns and mutates balances (CoinBalance >= 0, GemBalance >= 0).
Match aggregate exclusively finalizes match outcomes — only Match can mark a match FINISHED.
Cross-aggregate coordination uses domain events only; never direct aggregate mutation.
MatchFinished -> Wallet, MatchFinished -> Ranking, MissionCompleted -> Wallet.

---

## DEC-014: Code Generation and Layering Standard

status: accepted
scope: coding standards, repository pattern
source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md, J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md

Layer order: Handler -> Service -> Repository (Handler never calls Repository directly).
Only Service layer may initiate database transactions.
All public methods accept context.Context as first parameter.
sqlc used for all queries; no SELECT *; explicit column selection always.
All primary keys: UUID v7. All timestamps: UTC.
Code generation order: migrations -> sqlc queries -> repositories -> services -> handlers.

---

## DEC-015: Matchmaking Queue Key Schema

status: accepted
scope: Redis key naming, matchmaking
source: J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md

Queue key pattern: queue:{region}:{difficulty}:{stake} (Sorted Set, score = join timestamp).
Note: BACKEND_ARCHITECTURE.md uses queue:{difficulty}:{stake} (no region segment).
REDIS_SCHEMA.md is the more specific and later SPEC — its pattern takes precedence.
This discrepancy is logged as INFO auto-resolved.

---

## DEC-016: Admin Roles

status: accepted
scope: admin panel access control
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md

Four admin roles: SuperAdmin, GameAdmin, SupportAdmin, ReadOnlyAdmin.
Audit logs must record: admin login, user changes, economy changes, match actions.
