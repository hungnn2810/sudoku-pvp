# requirements.md

Functional requirements extracted from PRD (PRODUCT_REQUIREMENTS.MD) and
corroborated/extended by SPEC sources. Each entry includes source provenance.

source (PRD): J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
source (SPEC): J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
source (SPEC): J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
source (SPEC): J:\sources\sudoku-pvp\docs\API_CONTRACT.md
source (SPEC): J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
source (SPEC): J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md
source (existing): J:\sources\sudoku-pvp\.planning\REQUIREMENTS.md

---

## Authentication

REQ-auth-guest
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can log in as a guest without any credentials.
acceptance: POST /api/v1/auth/guest returns accessToken and refreshToken.

REQ-auth-google
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can log in using Google OAuth.
acceptance: POST /api/v1/auth/google completes Google OAuth flow and returns tokens.

REQ-auth-apple
source: J:\sources\sudoku-pvp\docs\OPENAPI.yaml
description: Players can log in using Apple sign-in (SPEC addition, not in PRD MVP scope).
acceptance: POST /api/v1/auth/apple completes Apple sign-in flow and returns tokens.
note: PRD MVP scope lists only Guest and Google. Apple login is present in OPENAPI.yaml
SPEC. Treated as SPEC extension — logged as INFO. No acceptance variant conflict
since PRD does not prohibit Apple, it simply omits it.

REQ-auth-refresh
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Clients can refresh an expired access token.
acceptance: POST /api/v1/auth/refresh returns a new accessToken given a valid refreshToken.

REQ-auth-jwt-middleware
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: All authenticated REST endpoints and the WebSocket handshake require a valid JWT Bearer token.
acceptance: Requests without valid JWT return 401. Token validated on every request.

---

## User Profile

REQ-user-profile-read
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Authenticated players can retrieve their own profile.
acceptance: GET /api/v1/me returns id, username, avatarUrl, level, rankTier, rankPoint, coinBalance.

REQ-user-profile-update
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Authenticated players can update their profile (username, avatar).
acceptance: PATCH /api/v1/me applies valid field updates and returns updated profile.

REQ-user-stats
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Players can view their gameplay statistics.
acceptance: GET /api/v1/me/stats returns win/loss/draw counts and match aggregate stats.

REQ-user-match-history
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Players can view their match history.
acceptance: GET /api/v1/matches returns paginated list of past matches. GET /api/v1/matches/{id} returns match detail.

---

## Wallet and Economy

REQ-wallet-read
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can view their current coin and gem balances.
acceptance: GET /api/v1/wallet returns coinBalance and gemBalance.

REQ-wallet-transactions
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can view their transaction history.
acceptance: GET /api/v1/wallet/transactions returns paginated ledger of all wallet transactions.

REQ-wallet-immutable-ledger
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md
description: Wallet transactions are never modified after creation. The ledger is append-only.
acceptance: No UPDATE or DELETE on wallet_transactions table. balance_before and balance_after stored on every record.

REQ-wallet-non-negative
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md
description: CoinBalance and GemBalance may never go below zero.
acceptance: Any operation that would reduce balance below 0 is rejected with an appropriate error before persistence.

REQ-wallet-aggregate-ownership
source: J:\sources\sudoku-pvp\docs\DDD_AGGREGATE_RULES.md
description: Only the Wallet aggregate may mutate balances.
acceptance: No direct balance mutations outside the Wallet service layer.

---

## Sudoku Puzzle

REQ-puzzle-retrieve
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Clients can retrieve a puzzle filtered by difficulty.
acceptance: GET /api/v1/sudoku/puzzle?difficulty={easy|medium|hard} returns puzzle grid without solution.

REQ-puzzle-no-solution-leak
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md
description: The solution grid is never transmitted to the client under any circumstance.
acceptance: solution_grid field is excluded from all API responses. Validated in integration tests.

---

## Game Modes

REQ-mode-solo
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can play a solo Sudoku session at Easy, Medium, or Hard difficulty.
acceptance: Player can start, pause, resume, use hint, and finish a solo game.

REQ-mode-bot-battle
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can play against a bot at Easy, Medium, or Hard difficulty.
acceptance: Bot fallback triggers on matchmaking queue timeout; bot match follows same battle engine rules.

REQ-mode-pvp
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can engage in real-time PvP matches with coin stake.
acceptance: Full matchmaking -> room -> battle -> result flow supported with coin stake.

---

## Matchmaking

REQ-matchmaking-join
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can join the matchmaking queue and cancel before a match is found.
acceptance: WebSocket events matchmaking.join and matchmaking.cancel handled. Player added to or removed from sorted set queue.

REQ-matchmaking-criteria
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Matching considers difficulty, stake, rank, and queue wait time.
acceptance: Matching worker applies rank filter, difficulty filter, and stake filter. Bot fallback on timeout.

REQ-matchmaking-slo
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Matchmaking completes within 10 seconds under normal load.
acceptance: P95 matchmaking time < 10 seconds. Measured in load tests.

---

## Room Lifecycle

REQ-room-lifecycle
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: A match progresses through the defined state machine before battle starts.
acceptance: States CREATED -> WAITING_PLAYERS -> READY -> COUNTDOWN (3s) -> PLAYING enforced server-side. room.ready, room.unready, room.updated, room.countdown events emitted correctly.

---

## Battle Engine

REQ-battle-move-submit
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: Players submit moves via WebSocket during PLAYING state.
acceptance: battle.submit_move accepted only in PLAYING state. 13-step validation pipeline executed server-side.

REQ-battle-move-validation-pipeline
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: Every move submission runs the full 13-step server-side validation pipeline.
acceptance: Steps: match exists, status=PLAYING, user in match, coords valid (0-8), cell editable, cell not complete, value valid (1-9), compare solution, score, combo, progress, completion checks, broadcast.

REQ-battle-scoring
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: Scores are calculated server-side per the defined scoring table with combo multipliers.
acceptance: Correct +10, Wrong -5+reset+lock(2s), Row +20, Column +20, Box +15, Puzzle +100. Combo 1.0x/1.2x/1.4x. All verified by unit tests.

REQ-battle-final-countdown
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: When first player completes puzzle, a 30-second final countdown begins for the opponent.
acceptance: battle.final_countdown event emitted with remainingSeconds=30. State transitions to FINAL_COUNTDOWN.

REQ-battle-winner-determination
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: Winner determined by score, then finish time, then draw.
acceptance: battle.ended event carries correct winner. Result persisted to match_players.

REQ-battle-surrender
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can surrender during a match.
acceptance: battle.surrender event triggers immediate loss with rank loss and coin loss for surrendering player.

REQ-battle-reconnect
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players who disconnect can reconnect within 30 seconds and restore match state.
acceptance: battle.reconnect_state event sent on reconnect with current score/combo/progress/timer. Auto-surrender after 30s grace period.

REQ-battle-hint
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can use hints during battle or solo play.
acceptance: battle.use_hint event handled; hint_used counter incremented; hint count tracked per match player.

REQ-battle-anti-cheat
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md
description: Rate limiting and anti-cheat rules enforced server-side for all moves.
acceptance: 5 moves/sec limit enforced. Violations: warning -> input lock -> forfeit. Client can never submit score, combo, or result. All enforced at boundary.

REQ-battle-slo-move
source: J:\sources\sudoku-pvp\docs\BATTLE_ENGINE_SPEC.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Move validation completes within 20ms at P95.
acceptance: P95 < 20ms measured under load. Broadcast P95 < 100ms.

---

## Ranking

REQ-ranking-my-rank
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can view their current rank tier and points.
acceptance: GET /api/v1/ranking/me returns tier and point.

REQ-ranking-leaderboard
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md, J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Players can view the global leaderboard.
acceptance: GET /api/v1/ranking/leaderboard returns ranked list ordered by rank_point DESC.

REQ-ranking-update-from-events
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md, J:\sources\sudoku-pvp\docs\EVENT_CATALOG.md
description: Ranking is updated from MatchFinished domain events.
acceptance: Ranking consumer on ranking.queue processes match.finished routing key and updates user rank_tier and rank_point.

REQ-ranking-tiers
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Seven rank tiers are supported.
acceptance: Valid tiers: Bronze, Silver, Gold, Platinum, Diamond, Master, Grandmaster. Promotion and demotion logic implemented.

---

## Missions

REQ-missions-list
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Players can view available daily and weekly missions and their progress.
acceptance: GET /api/v1/missions returns mission list with progress per user.

REQ-missions-claim
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Players can claim rewards for completed missions.
acceptance: POST /api/v1/missions/{id}/claim grants coinReward to wallet and marks mission claimed.

REQ-missions-event-driven
source: J:\sources\sudoku-pvp\docs\EVENT_CATALOG.md
description: Mission progress updated via domain events.
acceptance: mission.queue consumer processes relevant events and updates user_missions progress.

---

## Shop

REQ-shop-items
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Players can browse available shop items.
acceptance: GET /api/v1/shop/items returns list of active shop items with coin_price and gem_price.

REQ-shop-buy
source: J:\sources\sudoku-pvp\docs\API_CONTRACT.md
description: Players can purchase shop items.
acceptance: POST /api/v1/shop/buy with itemId deducts price from wallet, adds to inventory, publishes shop.purchased event.

REQ-shop-inventory-equip
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: Players can equip items from their inventory.
acceptance: Inventory tracks equipped state. At most one item of a type can be equipped at a time.

---

## WebSocket and Heartbeat

REQ-ws-heartbeat
source: J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md
description: WebSocket connections maintain a heartbeat to detect stale connections.
acceptance: Client sends battle.ping every 15s. Server responds battle.pong. Connection closed after 45s without ping.

REQ-ws-envelope
source: J:\sources\sudoku-pvp\docs\WEBSOCKET_PROTOCOL.md
description: All WebSocket messages use a standardized envelope format.
acceptance: Every message has { "event": "<name>", "data": {} } structure. Malformed envelopes rejected.

---

## Data and Infrastructure

REQ-postgres-schema
source: J:\sources\sudoku-pvp\docs\DATABASE_SCHEMA.md
description: PostgreSQL schema covers users, wallets, wallet_transactions, sudoku_puzzles, matches, match_players, match_moves, missions, user_missions, shop_items, inventory_items with defined indexes.
acceptance: Migrations create all tables and indexes as defined in DATABASE_SCHEMA.md.

REQ-redis-key-schema
source: J:\sources\sudoku-pvp\docs\REDIS_SCHEMA.md
description: Redis keys follow the module:resource:id convention with defined TTLs.
acceptance: match:{matchId}:state (2h), user:{userId}:connection (60s), queue:{region}:{difficulty}:{stake} (sorted set), rate:user:{userId}:move (1s), match:{matchId}:reconnect:{userId} (30s).

REQ-rabbitmq-topology
source: J:\sources\sudoku-pvp\docs\RABBITMQ_TOPOLOGY.md
description: RabbitMQ topology as defined: topic exchange game.events with five queues and dead-letter retry.
acceptance: Exchange, queues, bindings, and DLX provisioned on startup. Consumers active on all five queues.

REQ-domain-events-catalog
source: J:\sources\sudoku-pvp\docs\EVENT_CATALOG.md
description: All six domain events published and consumed per catalog: MatchStarted, MatchFinished, WalletTransactionCreated, RankChanged, MissionCompleted, ShopItemPurchased.
acceptance: Each event published with correct routing key and payload schema. Consumers idempotent.

---

## Quality and Testing

REQ-test-coverage
source: J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md, J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md
description: Minimum 80% unit test coverage for battle engine, ranking, and wallet units.
acceptance: Coverage gate enforced in CI. Test pyramid: 70% unit, 20% integration, 10% E2E.

REQ-integration-test-tooling
source: J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md
description: Integration tests use testcontainers-go for PostgreSQL, Redis, RabbitMQ, and WebSocket.
acceptance: All integration test suites pass in CI against real containers.

REQ-load-testing
source: J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md
description: Load tests cover 1000 concurrent matchmaking users, 500 active battles, and reconnect storms.
acceptance: Load tests run with k6 or vegeta. SLOs verified under load.

REQ-observability
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: OpenTelemetry tracing, Prometheus metrics, and Loki structured logs in all production paths.
acceptance: Trace IDs propagated. No fmt.Println in production code. Grafana dashboards operational.

---

## Admin Panel

REQ-admin-dashboard
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admin dashboard shows DAU, MAU, active matches, online users, and revenue.
acceptance: Dashboard data served via admin-authenticated endpoints.

REQ-admin-user-management
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admins can search, ban, unban, reset rank, and grant coins to users.
acceptance: All operations are gated by admin role. Results persisted and audit-logged.

REQ-admin-puzzle-management
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admins can upload, disable, and assign difficulty to puzzles.
acceptance: Puzzle operations reflected in sudoku_puzzles table and audit log.

REQ-admin-match-monitoring
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admins can view active matches, match details, and force-end matches.
acceptance: Force-end triggers match.finished event with admin actor noted in audit log.

REQ-admin-economy-management
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admins can grant or deduct coins and view transaction history.
acceptance: Economy actions go through Wallet aggregate. All operations audit-logged.

REQ-admin-mission-shop-management
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admins can create, edit, and disable missions and shop items.
acceptance: CRUD operations on missions and shop_items tables. Changes audit-logged.

REQ-admin-audit-logs
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: All privileged admin actions are recorded in an audit log.
acceptance: Audit log captures: admin login, user changes, economy changes, match actions. Immutable append-only records.

REQ-admin-roles
source: J:\sources\sudoku-pvp\docs\ADMIN_PANEL_SPEC.md
description: Admin access is role-gated with four defined roles.
acceptance: SuperAdmin, GameAdmin, SupportAdmin, ReadOnlyAdmin enforced by middleware. Unauthorized access returns 403.

---

## Non-Functional Requirements (from PRD and SPEC)

REQ-nfr-availability
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD, J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: System availability target: 99.9%.
acceptance: Measured over 30-day rolling window in production.

REQ-nfr-api-latency
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md
description: REST API P95 latency < 100ms.
acceptance: Measured via OpenTelemetry and Prometheus histograms.

REQ-nfr-retention
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: D1 retention >= 35%, D7 retention >= 15%.
acceptance: Measured via analytics events in analytics.queue consumer.

REQ-nfr-match-completion
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Match completion rate >= 90%.
acceptance: Tracked via match_finished vs match_started event ratio in analytics.

REQ-nfr-match-duration
source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD
description: Average match duration: 3-7 minutes.
acceptance: Tracked via match duration analytics.
