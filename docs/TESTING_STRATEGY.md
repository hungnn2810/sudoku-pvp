# TESTING_STRATEGY

## Pyramid

70% Unit Tests
20% Integration Tests
10% E2E Tests

## Unit

Battle Engine
Ranking
Wallet

Coverage >= 80%

## Integration

PostgreSQL
Redis
RabbitMQ
WebSocket

Use testcontainers-go

## Load Testing

Scenarios:

- 1000 concurrent matchmaking users
- 500 active battles
- reconnect storm

Tools:

k6
vegeta

## Critical Cases

- score calculation
- combo reset
- timeout
- reconnect
- rank update
