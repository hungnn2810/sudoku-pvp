---
phase: 1
plan: 03
subsystem: platform-foundation
tags: [redis, rabbitmq, infra, adapters, key-constants, topology, tdd]
dependency_graph:
  requires:
    - go-module-sudoku-pvp
    - project-directory-skeleton
  provides:
    - redis-client-factory
    - redis-key-constructors
    - rabbitmq-connection-recovery
    - rabbitmq-topology-provisioning
    - rabbitmq-publisher
    - rabbitmq-consumer
  affects:
    - all-phases-using-redis-state
    - all-phases-publishing-domain-events
    - all-phases-consuming-domain-events
tech_stack:
  added:
    - github.com/redis/go-redis/v9 v9.20.0
    - github.com/rabbitmq/amqp091-go v1.11.0
    - github.com/rs/zerolog v1.35.1
    - github.com/google/uuid v1.6.0
    - github.com/testcontainers/testcontainers-go v0.42.0
    - github.com/testcontainers/testcontainers-go/modules/rabbitmq v0.42.0
  patterns:
    - Redis key constructor functions centralizing all key formatting per REDIS_SCHEMA.md
    - RabbitMQ connection struct with sync.RWMutex protecting the amqp.Connection
    - NotifyClose-based reconnect goroutine with 5s retry delay and topology re-declaration
    - Per-goroutine channel allocation (channel-per-publisher, channel-per-consumer)
    - x-death header counting to enforce 3-retry DLX limit
    - TDD RED/GREEN cycle for both Redis keys and RabbitMQ integration tests
key_files:
  created:
    - internal/config/config.go
    - internal/redis/client.go
    - internal/redis/keys.go
    - internal/redis/keys_test.go
    - internal/rabbitmq/connection.go
    - internal/rabbitmq/topology.go
    - internal/rabbitmq/publisher.go
    - internal/rabbitmq/consumer.go
    - internal/rabbitmq/topology_test.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "Used goredis alias in redis/client.go to avoid package name collision with package redis"
  - "Connection.New() dials synchronously for fail-fast startup, then launches reconnectLoop goroutine"
  - "AMQP URL never logged in reconnectLoop — only connection events logged (T-03-01 mitigation)"
  - "DeclareAll called both at initial connect and after every successful reconnect for idempotent topology"
  - "Consumer.Consume uses select on ctx.Done() and delivery channel for goroutine leak prevention"
  - "x-death count >= 3 causes Ack (not Nack) to prevent infinite DLX retry loops"
metrics:
  duration: "8 minutes"
  completed: "2026-06-04T03:05:30Z"
  tasks_completed: 2
  tasks_total: 2
  files_created: 9
  files_modified: 2
---

# Phase 1 Plan 03: Redis and RabbitMQ Adapters Summary

**One-liner:** Redis go-redis/v9 client with 5 centralized key constructors and RabbitMQ amqp091-go adapter with NotifyClose reconnect loop, idempotent DeclareAll topology, Publisher, and Consumer with x-death retry guard.

## Tasks Completed

| Task | Name | Commits | Result |
|------|------|---------|--------|
| 1 | Redis client and key constants with unit tests | c9c994d (RED), b695af4 (GREEN) | All 6 unit tests pass |
| 2 | RabbitMQ connection recovery, topology, publisher, consumer | 8d8aeef (RED), a2168ad (GREEN) | All packages build; integration tests scaffold ready |

## What Was Built

### internal/config/config.go (blocker fix — plan 02 not yet executed)
Minimal config types `RedisConfig` and `RabbitMQConfig` needed to unblock redis/client.go compilation. Plan 02 will replace this with full Viper-backed loading.

### internal/redis/client.go
`NewClient(cfg config.RedisConfig) (*goredis.Client, error)` creates a go-redis/v9 client with production timeouts (dial 5s, read 3s, write 3s, min idle 5) and validates connectivity with Ping. Returns wrapped error on failure.

### internal/redis/keys.go
Five key constructor functions per REDIS_SCHEMA.md and DEC-015:
- `MatchStateKey(matchID)` → `match:{id}:state`
- `UserConnectionKey(userID)` → `user:{id}:connection`
- `QueueKey(region, difficulty, stake)` → `queue:{region}:{difficulty}:{stake}` (3-segment per DEC-015)
- `RateMoveKey(userID)` → `rate:user:{id}:move`
- `ReconnectKey(matchID, userID)` → `match:{matchId}:reconnect:{userId}`

### internal/redis/keys_test.go
Six unit tests covering exact string format for all 5 key functions plus a `TestKeyFormat_NoRawStrings` guard that verifies different UUIDs produce different keys. No network required.

### internal/rabbitmq/connection.go
`Connection` struct with `sync.RWMutex` guard on `*amqp.Connection`. `New()` dials synchronously, declares topology, then starts `reconnectLoop` goroutine. Loop watches `NotifyClose`, retries dial with 5s delay, re-declares topology on reconnect. AMQP URL is never logged.

### internal/rabbitmq/topology.go
`DeclareAll(ch)` idempotently provisions:
1. `game.events` topic exchange
2. `game.dlx` direct exchange
3. `game.dead.queue` bound to `game.dlx`
4. 5 domain queues (ranking, wallet, mission, analytics, notification) with `x-dead-letter-exchange=game.dlx`
5. Bindings: ranking/wallet → match.finished, mission → mission.completed, analytics → # (wildcard), notification → ranking.changed

### internal/rabbitmq/publisher.go
`Publisher` owns exactly one `*amqp.Channel` per DI contract. `Publish(ctx, routingKey, body)` calls `PublishWithContext` to `game.events` with `application/json` content type.

### internal/rabbitmq/consumer.go
`Consumer` owns exactly one `*amqp.Channel`. `Consume(ctx, handler)` selects on `ctx.Done()` and delivery channel. x-death count guard: if count >= 3, `Ack` and skip (prevents infinite DLX loop). Handler error → `Nack(requeue=false)` (routes to DLX). Handler success → `Ack`.

### internal/rabbitmq/topology_test.go
Integration test scaffold with `//go:build integration` tag using testcontainers-go rabbitmq module. Tests: ExchangeExists, AllQueuesExist, DLXExists, Idempotent, Publisher_Publish, Consumer_ReceivesMessage. Tests skip automatically if Docker is unavailable.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocker] Created minimal config package to unblock redis/client.go compilation**
- **Found during:** Task 1 setup
- **Issue:** Plan 03 depends on `config.RedisConfig` and `config.RabbitMQConfig` types from plan 02. Plan 02 had not been executed yet — the config directory only contained `.gitkeep`.
- **Fix:** Created `internal/config/config.go` with the two required struct types (`RedisConfig`, `RabbitMQConfig`) as specified in plan 03's `<interfaces>` section. Plan 02 will replace this with the full Viper-backed implementation.
- **Files modified:** `internal/config/config.go` (new)
- **Commit:** c9c994d

**2. [Rule 1 - Bug] Fixed external test package imports in topology_test.go**
- **Found during:** Task 2 GREEN verification
- **Issue:** Test file was in package `rabbitmq_test` but used exported symbols (`DeclareAll`, `ExchangeGameEvents`, `New`, etc.) without package qualification — undefined references on vet.
- **Fix:** Added `"sudoku-pvp/internal/rabbitmq"` import and prefixed all symbol references with `rabbitmq.`.
- **Files modified:** `internal/rabbitmq/topology_test.go`
- **Commit:** a2168ad

## Threat Mitigations Applied

| Threat | Mitigation Applied | File |
|--------|-------------------|------|
| T-03-01: AMQP URL in logs | `reconnectLoop` logs only connection events ("failed", "reconnected") never the URL | connection.go |
| T-03-02: Redis key collision | All 5 key constructors centralized; no raw strings in business code | keys.go |
| T-03-03: Goroutine leak from consumer | `Consume` selects on `ctx.Done()` and exits when context cancelled | consumer.go |
| T-03-04: Infinite DLX retry loop | x-death count >= 3 → Ack and discard instead of Nack | consumer.go |
| T-03-05: Shared amqp.Channel | `Channel()` creates new channel per call; Publisher and Consumer each own exactly one | connection.go, publisher.go, consumer.go |

## Known Stubs

None. All exported functions are fully implemented. The `internal/config/config.go` contains only type definitions (no stubs) and will be augmented by plan 02 without breaking these interfaces.

## Verification Results

| Check | Result |
|-------|--------|
| `go build ./internal/redis/...` | PASS |
| `go build ./internal/rabbitmq/...` | PASS |
| `go test ./internal/redis/... -run TestMatchStateKey -count=1` | PASS |
| `go test ./internal/redis/... -run TestQueueKey -count=1` | PASS |
| `go test ./internal/redis/... -run Test -count=1` (all 6 tests) | PASS |
| `go vet -tags integration ./internal/rabbitmq/...` | PASS |
| `go build -tags integration ./internal/rabbitmq/...` | PASS |
| `go.mod contains github.com/redis/go-redis/v9` | PASS |
| `go.mod contains github.com/rabbitmq/amqp091-go` | PASS |
| `go.mod contains github.com/rs/zerolog` | PASS |
| All 17 acceptance criteria checked | ALL PASS |

## Self-Check: PASSED

Files verified:
- internal/config/config.go: EXISTS
- internal/redis/client.go: EXISTS
- internal/redis/keys.go: EXISTS
- internal/redis/keys_test.go: EXISTS
- internal/rabbitmq/connection.go: EXISTS
- internal/rabbitmq/topology.go: EXISTS
- internal/rabbitmq/publisher.go: EXISTS
- internal/rabbitmq/consumer.go: EXISTS
- internal/rabbitmq/topology_test.go: EXISTS

Commits verified:
- c9c994d: test(01-03): add failing tests for Redis key constructors
- b695af4: feat(01-03): implement Redis client factory and key constructors
- 8d8aeef: test(01-03): add failing integration tests for RabbitMQ topology
- a2168ad: feat(01-03): implement RabbitMQ connection recovery, topology, publisher, and consumer
