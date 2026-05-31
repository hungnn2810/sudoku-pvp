# DDD_AGGREGATE_RULES

## Aggregates

User
Wallet
Match
Mission
ShopItem

## Rules

### User
Owns profile and ranking metadata.

### Wallet
Owns balances.
All balance changes go through Wallet aggregate.

### Match
Owns:
- players
- score
- progress
- result

Only Match aggregate can finish a match.

### Mission
Owns mission progress and rewards.

## Cross Aggregate Rules

MatchFinished -> Wallet
MatchFinished -> Ranking
MissionCompleted -> Wallet

Use domain events.
Never update multiple aggregates directly.
