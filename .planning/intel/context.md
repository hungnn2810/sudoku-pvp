# context.md

Background and domain context from DOC-type sources, supplemented by SPEC background sections.
Entries keyed by topic with source attribution.

---

## Topic: Project Identity and Vision

source: J:\sources\sudoku-pvp\docs\PRODUCT_REQUIREMENTS.MD (PRD)
source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC)

Sudoku Battle Arena is a competitive real-time Sudoku game. Players can engage
in Solo play, Bot Battles, and real-time PvP with rank progression. The product
targets short sessions, addictive gameplay, and a not-pay-to-win economy.

Target personas:
- Casual Player: solve Sudoku, pass time.
- Competitive Player: climb ranks, challenge others.
- Collector: unlock skins, collect avatars, complete achievements.

Product goals: short sessions, PvP tension, not pay-to-win.
MVP excludes: Clan, Tournament, Spectator, Replay, Battle Pass, Guild, Chat.

---

## Topic: Domain-Driven Design Structure

source: J:\sources\sudoku-pvp\docs\DDD_AGGREGATE_RULES.md (DOC, medium confidence)
source: J:\sources\sudoku-pvp\docs\DOMAIN_MODEL.md (SPEC)

Core Domain aggregates: Battle, Matchmaking, Ranking, Wallet.
Supporting Domain aggregates: User, Mission, Shop, Analytics.

Aggregate boundaries:
- User: owns profile and ranking metadata.
- Wallet: owns balances exclusively. All balance mutations route through Wallet aggregate.
- Match: owns players, score, progress, and result. Only Match can finalize a match.
- Mission: owns mission progress and reward state.
- ShopItem: owns shop catalog entries.

Cross-aggregate communication uses domain events only via RabbitMQ.
Never update multiple aggregates directly in a single transaction.
Event chains: MatchFinished -> Wallet, MatchFinished -> Ranking, MissionCompleted -> Wallet.

---

## Topic: AI Code Generation Context

source: J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md (DOC)

This document configures AI assistant behavior for code generation in the project.
Key rules relevant to code generation context:
- Follow BACKEND_ARCHITECTURE.md, DOMAIN_MODEL.md, DATABASE_SCHEMA.md, EVENT_CATALOG.md, BATTLE_ENGINE_SPEC.md.
- Note: MATCHMAKING_SPEC.md is referenced but no classification file was found for it in this
  ingest set — it may be a planned but not-yet-authored document.
- Context-first parameters, no SELECT *, no business logic in handlers, repository required.
- OpenTelemetry tracing, UTC timestamps, UUID v7.

---

## Topic: Backend Implementation Task Phases

source: J:\sources\sudoku-pvp\docs\ENGINE_TASKS.md (DOC)

The project is organized into 16 implementation phases (0-15):
Phase 0: Foundation (infra, Docker, DB, Redis, RabbitMQ, migrations, sqlc, logger, OTel)
Phase 1: Authentication (JWT, guest, Google, middleware)
Phase 2: User (profile, stats, avatar)
Phase 3: Wallet (balance, transactions, deposit, withdraw)
Phase 4: Sudoku (puzzle table, repository, difficulty, API)
Phase 5: Matchmaking (Redis queue, workers, rank/difficulty/stake filters, bot fallback)
Phase 6: Room (create, join, ready, countdown)
Phase 7: WebSocket Gateway (connection manager, user registry, event router, auth, broadcast, reconnect)
Phase 8: Battle Engine (validation, scoring, combo, progress, row/col/box, puzzle completion, winner, final countdown, end match)
Phase 9: Ranking (rank calc, promotion, demotion, leaderboard)
Phase 10: Missions (daily, weekly, progress, claim)
Phase 11: Shop (items, buy, inventory, equip)
Phase 12: Analytics (event collection, match/economy/retention analytics)
Phase 13: Anti-Cheat (rate limiting, move spam detection, time validation, suspicious activity logging)
Phase 14: Production (Docker image, CI/CD, Grafana, Loki, Prometheus)
Phase 15: Testing (unit: Battle Engine/Ranking/Wallet; integration: PG/Redis/RabbitMQ/WS; load: matchmaking/battle/gateway)

---

## Topic: Coding Conventions and Standards

source: J:\sources\sudoku-pvp\docs\CODING_STANDARDS.md (DOC)

Package structure per module: handler.go, service.go, repository.go, model.go, dto.go, errors.go.
Interface naming: UserRepository (interface), PostgresUserRepository (implementation).
DTO naming: LoginRequest (request), LoginResponse (response).
WebSocket event format: { "event": "battle.started", "data": {} }.
Redis key pattern: module:resource:id (e.g., match:123:state, user:123:connection).
Typed errors required (var ErrUserNotFound = errors.New("user not found")).
PR checklist: unit test, integration test, no SELECT *, no business logic in handler/repo,
structured logging, OpenTelemetry tracing, context propagated.

---

## Topic: Testing Strategy

source: J:\sources\sudoku-pvp\docs\TESTING_STRATEGY.md (DOC)

Test pyramid: 70% unit, 20% integration, 10% E2E.
Unit test targets: battle engine, ranking, wallet (>= 80% coverage).
Integration test tooling: testcontainers-go for PostgreSQL, Redis, RabbitMQ, WebSocket.
Load testing tools: k6, vegeta.
Load scenarios: 1000 concurrent matchmaking users, 500 active battles, reconnect storm.
Critical test cases: score calculation, combo reset, timeout, reconnect, rank update.

---

## Topic: Future Scaling Roadmap (Post-MVP)

source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC)

Phase 1 (current): Modular Monolith.
Phase 2: Extract Matchmaking Service and Battle Service.
Phase 3: Extract Wallet Service and Ranking Service.
Phase 4: Multi-region deployment.
Battle Engine must remain compatible with future extraction.
Reserved features (not in MVP): Tournament Mode, Spectator Mode, Replay System,
Power Ups, Team Battle, Clan Battle.

---

## Topic: Deployment Topology (Minimum Production)

source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC)

Minimum production setup:
- 3 API Pods (horizontal scaling)
- 1 PostgreSQL Primary
- 1 Redis instance
- 1 RabbitMQ instance
- 1 Nginx Ingress

---

## Topic: Analytics Event Catalog

source: J:\sources\sudoku-pvp\docs\BACKEND_ARCHITECTURE.md (SPEC)

Gameplay events: match_started, match_finished, move_submitted, hint_used, player_surrendered.
Economy events: coin_earned, coin_spent, shop_purchase.
Retention events: daily_login, daily_mission_completed.
All analytics events routed to analytics.queue via game.events exchange.

---

## Topic: Missing Reference — MATCHMAKING_SPEC.md

source: J:\sources\sudoku-pvp\docs\CLAUDE_CODE_MASTER_PROMPT.md (DOC cross_ref)

The CLAUDE_CODE_MASTER_PROMPT.md references docs/MATCHMAKING_SPEC.md as a required
follow document, but no classification JSON for MATCHMAKING_SPEC.md was present in
the CLASSIFICATIONS_DIR for this ingest. This document may be planned but not yet
authored, or may have been excluded from the ingest manifest. Matchmaking behavior
is covered by BACKEND_ARCHITECTURE.md Section 5 and ENGINE_TASKS.md Phase 5, but a
dedicated spec may be needed for complete matchmaking contract coverage.
