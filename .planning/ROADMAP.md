# ROADMAP.md

## Milestone 1: MVP Backend Foundation

### Phase 1: Platform Foundation
- Project structure, config, and environment setup.
- PostgreSQL/Redis/RabbitMQ wiring.
- Migration tool + sqlc setup.
- Structured logger + tracing bootstrap.

### Phase 2: Auth & Identity
- Guest login + Google login.
- JWT issue/refresh/validate.
- Auth middleware for REST and WebSocket handshake.

### Phase 3: User & Wallet Core
- User profile query/update flow.
- Wallet read model and transaction history.
- Wallet transaction command handling with invariants.

### Phase 4: Puzzle & Battle Engine
- Puzzle repository and difficulty selection.
- Authoritative move validation + scoring + combo logic.
- Match progression, timeout, and deterministic result resolution.

### Phase 5: Matchmaking & Realtime
- Queue join/cancel and matching workers.
- WebSocket realtime protocol and room lifecycle.
- Reconnect and heartbeat enforcement.

### Phase 6: Ranking, Missions, Shop, Events
- Event publisher/subscriber flows on RabbitMQ.
- Ranking update pipeline from match finished events.
- Mission and shop integration with wallet effects.

### Phase 7: Admin & Hardening
- Admin APIs/features for operations.
- Audit logs for privileged actions.
- Load/concurrency testing, smoke suites, and rollback notes.
