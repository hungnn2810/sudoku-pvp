# REQUIREMENTS.md

Synthesized from 18 classified documents via gsd-ingest-docs (merge mode).
Precedence applied: ADR > SPEC > PRD > DOC.
Source provenance retained per requirement.

---

## Authentication

- REQ-001: Support guest login and Google login for player access.
- REQ-auth-guest: POST /api/v1/auth/guest returns accessToken and refreshToken. [PRD]
- REQ-auth-google: POST /api/v1/auth/google completes Google OAuth flow and returns tokens. [PRD]
- REQ-auth-apple: POST /api/v1/auth/apple completes Apple sign-in flow and returns tokens. [SPEC/OPENAPI — not in PRD MVP scope; confirm with product owner before deprioritizing]
- REQ-auth-refresh: POST /api/v1/auth/refresh returns new accessToken given a valid refreshToken. [SPEC]
- REQ-auth-jwt-middleware: All authenticated REST endpoints and the WebSocket handshake require a valid JWT Bearer token. Requests without valid JWT return 401. [SPEC]

## User Profile

- REQ-007: Implement documented endpoints for auth, profile, wallet, and wallet transaction history.
- REQ-user-profile-read: GET /api/v1/me returns id, username, avatarUrl, level, rankTier, rankPoint, coinBalance. [SPEC]
- REQ-user-profile-update: PATCH /api/v1/me applies valid field updates and returns updated profile. [SPEC]
- REQ-user-stats: GET /api/v1/me/stats returns win/loss/draw counts and match aggregate stats. [SPEC]
- REQ-user-match-history: GET /api/v1/matches returns paginated past matches. GET /api/v1/matches/{id} returns match detail. [SPEC]

## Wallet and Economy

- REQ-016: Wallet aggregate exclusively owns and mutates balances.
- REQ-017: Track wallet transactions with immutable audit trail.
- REQ-wallet-read: GET /api/v1/wallet returns coinBalance and gemBalance. [PRD, SPEC]
- REQ-wallet-transactions: GET /api/v1/wallet/transactions returns paginated ledger of all wallet transactions. [PRD, SPEC]
- REQ-wallet-immutable-ledger: No UPDATE or DELETE on wallet_transactions table. balance_before and balance_after stored on every record. [SPEC]
- REQ-wallet-non-negative: CoinBalance and GemBalance may never go below zero. Any operation reducing balance below 0 is rejected before persistence. [SPEC]
- REQ-wallet-aggregate-ownership: No direct balance mutations outside the Wallet service layer. [SPEC/DDD]

## Sudoku Puzzle

- REQ-puzzle-retrieve: GET /api/v1/sudoku/puzzle?difficulty={easy|medium|hard} returns puzzle grid without solution. [SPEC]
- REQ-puzzle-no-solution-leak: The solution grid is NEVER transmitted to the client under any circumstance. Validated in integration tests. [SPEC — security critical]

## Game Modes

- REQ-002: Provide solo, bot battle, and realtime PvP modes as defined by MVP docs.
- REQ-mode-solo: Player can start, pause, resume, use hint, and finish a solo game at Easy/Medium/Hard difficulty. [PRD]
- REQ-mode-bot-battle: Bot fallback triggers on matchmaking queue timeout. Bot match follows same battle engine rules. [PRD]
- REQ-mode-pvp: Full matchmaking -> room -> battle -> result flow with coin stake. [PRD]

## Matchmaking

- REQ-013: Provide queue join/cancel and matching by rank/difficulty constraints.
- REQ-matchmaking-join: WebSocket events matchmaking.join and matchmaking.cancel handled. Player added to or removed from sorted set queue. [SPEC]
- REQ-matchmaking-criteria: Matching worker applies rank filter, difficulty filter, and stake filter. Bot fallback on timeout. Queue key: queue:{region}:{difficulty}:{stake}. [SPEC]
- REQ-matchmaking-slo: P95 matchmaking time < 10 seconds. Measured in load tests. [PRD, SPEC]

## Room Lifecycle

- REQ-014: Persist active match state in Redis with TTL and reconnect metadata.
- REQ-room-lifecycle: States CREATED -> WAITING_PLAYERS -> READY -> COUNTDOWN (3s) -> PLAYING enforced server-side. room.ready, room.unready, room.updated, room.countdown events emitted correctly. [SPEC]

## Battle Engine

- REQ-003: Enforce server-authoritative battle outcomes; client data is never trusted for scoring, combo, rewards, ranking, or result.
- REQ-004: Ensure deterministic battle computation: same input yields same output.
- REQ-015: Only Match aggregate may finalize match outcomes.
- REQ-battle-move-submit: battle.submit_move accepted only in PLAYING state. [SPEC]
- REQ-battle-move-validation-pipeline: Every move submission runs the full 13-step server-side validation pipeline: match exists, status=PLAYING, user in match, coords valid (0-8), cell editable, cell not complete, value valid (1-9), compare solution, score, combo, progress, completion checks, broadcast. [SPEC]
- REQ-battle-scoring: Correct +10, Wrong -5+reset+lock(2s), Row +20, Column +20, Box +15, Puzzle +100. Combo: 1x/1.2x/1.4x. All verified by unit tests. [SPEC]
- REQ-battle-final-countdown: When first player completes puzzle, battle.final_countdown event emitted with remainingSeconds=30. State transitions to FINAL_COUNTDOWN. [SPEC]
- REQ-battle-winner-determination: Winner by score, then finish time, then draw. battle.ended event carries correct winner. Result persisted to match_players. [SPEC]
- REQ-battle-surrender: battle.surrender triggers immediate loss with rank loss and coin loss for surrendering player. [SPEC, PRD]
- REQ-battle-reconnect: Players who disconnect can reconnect within 30 seconds and restore match state via battle.reconnect_state event (score/combo/progress/timer). Auto-surrender after 30s grace period. [SPEC, PRD]
- REQ-battle-hint: battle.use_hint event handled; hint_used counter incremented; hint count tracked per match player. [SPEC, PRD]
- REQ-battle-anti-cheat: 5 moves/sec limit enforced. Violations: warning -> input lock -> forfeit. Client can never submit score, combo, or result. [SPEC]
- REQ-battle-slo-move: P95 move validation < 20ms. P95 broadcast < 100ms. Measured under load. [SPEC, PRD]

## Ranking

- REQ-018: Update ranking from match outcomes via domain/event flow.
- REQ-ranking-my-rank: GET /api/v1/ranking/me returns tier and point. [SPEC, PRD]
- REQ-ranking-leaderboard: GET /api/v1/ranking/leaderboard returns ranked list ordered by rank_point DESC. [SPEC, PRD]
- REQ-ranking-update-from-events: Ranking consumer on ranking.queue processes match.finished routing key and updates user rank_tier and rank_point. [SPEC]
- REQ-ranking-tiers: Valid tiers: Bronze, Silver, Gold, Platinum, Diamond, Master, Grandmaster. Promotion and demotion logic implemented. [SPEC]

## Missions

- REQ-019: Support mission progression and reward grant through event-driven processing.
- REQ-missions-list: GET /api/v1/missions returns mission list with progress per user. [SPEC]
- REQ-missions-claim: POST /api/v1/missions/{id}/claim grants coinReward to wallet and marks mission claimed. [SPEC]
- REQ-missions-event-driven: mission.queue consumer processes relevant events and updates user_missions progress. [SPEC]

## Shop

- REQ-020: Support shop item lifecycle and purchase-related eventing.
- REQ-shop-items: GET /api/v1/shop/items returns list of active shop items with coin_price and gem_price. [SPEC]
- REQ-shop-buy: POST /api/v1/shop/buy with itemId deducts price from wallet, adds to inventory, publishes shop.purchased event. [SPEC]
- REQ-shop-inventory-equip: Inventory tracks equipped state. At most one item of a type can be equipped at a time. [SPEC]

## WebSocket and Heartbeat

- REQ-009: Provide WebSocket connection endpoint `/ws/connect` with JWT auth.
- REQ-010: Support documented client/server event envelopes and battle/matchmaking event types.
- REQ-011: Implement heartbeat flow (`battle.ping`/`battle.pong`) with timeout handling.
- REQ-012: Support reconnect state recovery for active matches.
- REQ-ws-heartbeat: Client sends battle.ping every 15s. Server responds battle.pong. Connection closed after 45s without ping. [SPEC]
- REQ-ws-envelope: Every message has { "event": "<name>", "data": {} } structure. Malformed envelopes rejected. [SPEC]

## Data and Infrastructure

- REQ-021: Implement PostgreSQL schema for users, wallets, transactions, matches, and puzzle assets.
- REQ-022: Implement Redis key conventions and TTL policies for match, user connection, queue, rate limits, reconnect.
- REQ-023: Implement RabbitMQ topology using topic exchange `game.events` and defined routing keys.
- REQ-024: Publish and consume domain events per event catalog.
- REQ-postgres-schema: Migrations create all tables and indexes per DATABASE_SCHEMA.md: users, wallets, wallet_transactions, sudoku_puzzles, matches, match_players, match_moves, missions, user_missions, shop_items, inventory_items. [SPEC]
- REQ-redis-key-schema: Redis keys: match:{matchId}:state (2h), user:{userId}:connection (60s), queue:{region}:{difficulty}:{stake} (sorted set), rate:user:{userId}:move (1s), match:{matchId}:reconnect:{userId} (30s). [SPEC]
- REQ-rabbitmq-topology: Exchange game.events (topic), five queues, DLX retry with 3 attempts, dead letter queue game.dead.queue. Provisioned on startup. [SPEC]
- REQ-domain-events-catalog: Six domain events published and consumed per catalog: MatchStarted, MatchFinished, WalletTransactionCreated, RankChanged, MissionCompleted, ShopItemPurchased. Each consumer idempotent. [SPEC]

## Quality and Testing

- REQ-025: Follow coding standards: handler->service->repository layering and context-first public methods.
- REQ-026: Enforce structured logging only in production paths.
- REQ-027: Provide unit, integration, and smoke test coverage for critical match/scoring behavior.
- REQ-028: Reach at least 80% coverage for battle, ranking, and wallet units per testing strategy.
- REQ-test-coverage: Coverage gate enforced in CI. Test pyramid: 70% unit, 20% integration, 10% E2E. [DOC]
- REQ-integration-test-tooling: Integration tests use testcontainers-go for PostgreSQL, Redis, RabbitMQ, and WebSocket. All suites pass in CI against real containers. [DOC]
- REQ-load-testing: Load tests cover 1000 concurrent matchmaking users, 500 active battles, and reconnect storms. Run with k6 or vegeta. SLOs verified under load. [DOC]
- REQ-observability: OpenTelemetry tracing, Prometheus metrics, and Loki structured logs in all production paths. Trace IDs propagated. No fmt.Println in production code. [SPEC]

## Admin Panel

- REQ-029: Provide admin operations for user, puzzle, match, economy, mission, and shop management.
- REQ-030: Record admin audit logs for admin login and all privileged state changes.
- REQ-admin-dashboard: Dashboard data served via admin-authenticated endpoints: DAU, MAU, active matches, online users, revenue. [SPEC]
- REQ-admin-user-management: Admins can search, ban, unban, reset rank, and grant coins. All operations role-gated and audit-logged. [SPEC]
- REQ-admin-puzzle-management: Admins can upload, disable, and assign difficulty to puzzles. Operations reflected in sudoku_puzzles and audit log. [SPEC]
- REQ-admin-match-monitoring: Admins can view active matches and force-end matches. Force-end triggers match.finished event with admin actor in audit log. [SPEC]
- REQ-admin-economy-management: Admin coin grants/deductions go through Wallet aggregate. All operations audit-logged. [SPEC]
- REQ-admin-mission-shop-management: CRUD on missions and shop_items tables. Changes audit-logged. [SPEC]
- REQ-admin-audit-logs: Audit log captures: admin login, user changes, economy changes, match actions. Immutable append-only records. [SPEC]
- REQ-admin-roles: Four roles: SuperAdmin, GameAdmin, SupportAdmin, ReadOnlyAdmin enforced by middleware. Unauthorized access returns 403. [SPEC]

## Non-Functional Requirements

- REQ-nfr-availability: System availability 99.9%. Measured over 30-day rolling window. [PRD, SPEC]
- REQ-nfr-api-latency: REST API P95 latency < 100ms. Measured via OpenTelemetry and Prometheus histograms. [SPEC]
- REQ-nfr-retention: D1 retention >= 35%, D7 retention >= 15%. Measured via analytics.queue consumer. [PRD]
- REQ-nfr-match-completion: Match completion rate >= 90%. Tracked via match_finished vs match_started ratio. [PRD]
- REQ-nfr-match-duration: Average match duration 3-7 minutes. Tracked via match duration analytics. [PRD]
