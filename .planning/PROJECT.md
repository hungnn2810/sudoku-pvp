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
