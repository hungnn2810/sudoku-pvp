# REQUIREMENTS.md

## Product & Gameplay
- REQ-001: Support guest login and Google login for player access.
- REQ-002: Provide solo, bot battle, and realtime PvP modes as defined by MVP docs.
- REQ-003: Enforce server-authoritative battle outcomes; client data is never trusted for scoring, combo, rewards, ranking, or result.
- REQ-004: Ensure deterministic battle computation: same input yields same output.

## API & Contracts
- REQ-005: Expose versioned REST API under `/api/v1`.
- REQ-006: Support authenticated endpoints with Bearer JWT.
- REQ-007: Implement documented endpoints for auth, profile, wallet, and wallet transaction history.
- REQ-008: Keep APIs idempotent and deterministic where applicable.

## Realtime
- REQ-009: Provide WebSocket connection endpoint `/ws/connect` with JWT auth.
- REQ-010: Support documented client/server event envelopes and battle/matchmaking event types.
- REQ-011: Implement heartbeat flow (`battle.ping`/`battle.pong`) with timeout handling.
- REQ-012: Support reconnect state recovery for active matches.

## Matchmaking & Match Flow
- REQ-013: Provide queue join/cancel and matching by rank/difficulty constraints.
- REQ-014: Persist active match state in Redis with TTL and reconnect metadata.
- REQ-015: Only Match aggregate may finalize match outcomes.

## Economy, Ranking, Missions, Shop
- REQ-016: Wallet aggregate exclusively owns and mutates balances.
- REQ-017: Track wallet transactions with immutable audit trail.
- REQ-018: Update ranking from match outcomes via domain/event flow.
- REQ-019: Support mission progression and reward grant through event-driven processing.
- REQ-020: Support shop item lifecycle and purchase-related eventing.

## Data & Events
- REQ-021: Implement PostgreSQL schema for users, wallets, transactions, matches, and puzzle assets.
- REQ-022: Implement Redis key conventions and TTL policies for match, user connection, queue, rate limits, reconnect.
- REQ-023: Implement RabbitMQ topology using topic exchange `game.events` and defined routing keys.
- REQ-024: Publish and consume domain events per event catalog.

## Quality & Operations
- REQ-025: Follow coding standards: handler->service->repository layering and context-first public methods.
- REQ-026: Enforce structured logging only in production paths.
- REQ-027: Provide unit, integration, and smoke test coverage for critical match/scoring behavior.
- REQ-028: Reach at least 80% coverage for battle, ranking, and wallet units per testing strategy.
- REQ-029: Provide admin operations for user, puzzle, match, economy, mission, and shop management.
- REQ-030: Record admin audit logs for admin login and all privileged state changes.
