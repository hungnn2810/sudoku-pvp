# Sudoku Battle Arena - Backend Architecture

Version: 1.0

Target Audience:

* Backend Engineers
* Claude Code
* OpenAI Codex
* Cursor
* DevOps Engineers

---

# 1. Project Goal

Build a scalable backend for a real-time Sudoku PvP game.

Core requirements:

* Real-time PvP battles
* Matchmaking
* Ranking
* Coin economy
* Daily missions
* Shop
* Analytics
* Anti-cheat

Backend must be server-authoritative.

Clients are never trusted for:

* score
* combo
* rewards
* ranking
* battle results
* move validation

All gameplay decisions are calculated by the server.

---

# 2. Architecture Philosophy

## MVP First

Do not build microservices initially.

Use:

* Modular Monolith
* Event Driven Internal Architecture
* Domain Oriented Modules

Benefits:

* Faster development
* Easier deployment
* Easier debugging
* Lower infrastructure cost

The architecture must allow future extraction into microservices.

---

# 3. Technology Stack

## Language

Go 1.25+

---

## HTTP Framework

Gin

---

## Database

PostgreSQL 17+

---

## Query Layer

sqlc

Avoid:

* GORM
* Generic Repository

---

## Cache

Redis 8+

Used for:

* Matchmaking queue
* Online users
* Realtime battle state
* Rate limiting
* Session cache
* Reconnect state

---

## Realtime

WebSocket

Recommended library:

nhooyr/websocket

---

## Messaging

RabbitMQ

Used for:

* Analytics
* Mission processing
* Notifications
* Wallet events
* Match completion events

---

## Observability

OpenTelemetry

Metrics:

* Prometheus

Logs:

* Loki

Dashboards:

* Grafana

---

## Deployment

Docker

---

# 4. Repository Structure

```text
backend/

cmd/
└── api/
    └── main.go

internal/

├── auth/
├── user/
├── wallet/
├── sudoku/
├── matchmaking/
├── room/
├── battle/
├── ranking/
├── mission/
├── shop/
├── analytics/
├── admin/

├── common/
│
├── config/
├── database/
├── redis/
├── rabbitmq/
├── websocket/
├── middleware/
├── logger/
├── telemetry/

pkg/

migrations/

deployments/
├── docker/

scripts/

docs/
```

---

# 5. Domain Modules

## Auth Module

Responsibilities:

* Guest login
* Google login
* Apple login
* JWT generation
* Refresh token

Endpoints:

```http
POST /auth/guest
POST /auth/google
POST /auth/apple
POST /auth/refresh
```

---

## User Module

Responsibilities:

* Profile
* Statistics
* Avatar

Endpoints:

```http
GET /me
PATCH /me
GET /me/stats
GET /me/history
```

---

## Wallet Module

Responsibilities:

* Coin balance
* Gem balance
* Transaction history

Rules:

* Ledger based
* Immutable transactions
* Never modify historical records

Endpoints:

```http
GET /wallet
GET /wallet/transactions
```

---

## Sudoku Module

Responsibilities:

* Puzzle retrieval
* Puzzle generation
* Difficulty classification

Rules:

* Puzzle sent to client
* Solution never sent to client

Endpoints:

```http
GET /sudoku/puzzle
```

---

## Matchmaking Module

Responsibilities:

* Queue management
* Opponent selection
* Bot fallback

Matching criteria:

* Difficulty
* Stake
* Rank
* Queue waiting time

---

## Room Module

Responsibilities:

* Lobby
* Ready state
* Countdown

Lifecycle:

```text
CREATED

WAITING

READY

COUNTDOWN

STARTED

FINISHED
```

---

## Battle Module

Most critical module.

Responsibilities:

* Move validation
* Score calculation
* Combo calculation
* Win/loss determination
* Realtime synchronization

All battle logic belongs here.

---

## Ranking Module

Responsibilities:

* Rank point calculation
* Tier progression
* Leaderboard

Ranks:

```text
Bronze
Silver
Gold
Platinum
Diamond
Master
Grandmaster
```

---

## Mission Module

Responsibilities:

* Daily missions
* Weekly missions
* Achievements

---

## Shop Module

Responsibilities:

* Cosmetic purchases
* Inventory management
* Item equipping

---

# 6. Database Design

## users

```sql
id
username
avatar_url
level
exp
rank_tier
rank_point

created_at
updated_at
```

---

## wallets

```sql
id
user_id

coin_balance
gem_balance

updated_at
```

---

## wallet_transactions

```sql
id
user_id

type
amount

balance_before
balance_after

reference_type
reference_id

created_at
```

---

## sudoku_puzzles

```sql
id

difficulty

puzzle_grid
solution_grid

empty_count

created_at
```

---

## matches

```sql
id

mode
difficulty

stake_coin

status

puzzle_id

started_at
ended_at

created_at
```

---

## match_players

```sql
id

match_id
user_id

score
progress

combo

wrong_count

hint_used

result

coin_change
rank_change

finished_at
```

---

## match_moves

```sql
id

match_id
user_id

row_index
col_index

value

is_correct

score_delta

created_at
```

---

# 7. Redis Design

## Online User

```text
user:{userId}:connection
```

TTL:

```text
60 seconds
```

---

## Match State

```text
match:{matchId}:state
```

TTL:

```text
2 hours
```

Example:

```json
{
  "match_id":"123",
  "status":"playing",
  "players":{
    "u1":{
      "score":100,
      "combo":2,
      "progress":45
    },
    "u2":{
      "score":80,
      "combo":1,
      "progress":40
    }
  }
}
```

---

## Matchmaking Queue

```text
queue:{difficulty}:{stake}
```

Example:

```text
queue:medium:100
```

Redis Sorted Set Score:

```text
join_timestamp
```

---

# 8. Battle Engine

Battle Engine is server authoritative.

---

## Validation Flow

```text
1. Match exists

2. Match status = PLAYING

3. User belongs to match

4. Cell coordinates valid

5. Cell not already completed

6. Compare submitted value with solution

7. Calculate score

8. Update combo

9. Update progress

10. Check row completion

11. Check column completion

12. Check box completion

13. Check puzzle completion

14. Persist move

15. Broadcast update
```

---

## Score Rules

```text
Correct Move      +10

Wrong Move        -5

Complete Row      +20

Complete Column   +20

Complete Box      +15

Combo x2          +20%

Combo x3          +40%

Puzzle Complete   +100
```

---

# 9. WebSocket Architecture

Connection endpoint:

```http
GET /ws/connect
```

Authentication:

```text
Bearer JWT
```

---

## Client Events

```text
matchmaking.join

matchmaking.cancel

room.ready

room.unready

battle.submit_move

battle.use_hint

battle.surrender

battle.ping
```

---

## Server Events

```text
matchmaking.found

room.updated

room.countdown

battle.started

battle.move_result

battle.opponent_update

battle.final_countdown

battle.ended

battle.error

battle.reconnect_state
```

---

# 10. Anti Cheat

Rule 1

Client never submits score.

---

Rule 2

Client never submits combo.

---

Rule 3

Client never submits battle result.

---

Rule 4

Client never receives solution.

---

Rule 5

Move rate limit:

```text
5 moves / second
```

Violation:

```text
warning

temporary block

match forfeit
```

---

# 11. Reconnect Handling

Reconnect window:

```text
30 seconds
```

Flow:

```text
disconnect

↓

keep match alive

↓

user reconnect

↓

restore state from Redis

↓

continue battle
```

---

# 12. Event Driven Architecture

## MatchStarted

Published by:

Battle Module

Consumers:

* Analytics

---

## MatchFinished

Published by:

Battle Module

Consumers:

* Wallet
* Ranking
* Mission
* Analytics

---

## WalletTransactionCreated

Published by:

Wallet Module

Consumers:

* Analytics
* Notification

---

# 13. Analytics Events

Gameplay:

```text
match_started

match_finished

move_submitted

hint_used

player_surrendered
```

Economy:

```text
coin_earned

coin_spent

shop_purchase
```

Retention:

```text
daily_login

daily_mission_completed
```

---

# 14. Deployment

Minimum Production Setup

```text
3 API Pods

1 PostgreSQL Primary

1 Redis

1 RabbitMQ

1 Nginx Ingress
```

---

# 15. Non Functional Requirements

API P95 Latency

```text
< 100ms
```

Battle Move Validation

```text
< 20ms
```

WebSocket Broadcast

```text
< 100ms
```

Matchmaking Time

```text
< 10 seconds
```

Availability

```text
99.9%
```

---

# 16. Future Scaling Plan

Phase 1

Modular Monolith

---

Phase 2

Extract:

* Matchmaking Service
* Battle Service

---

Phase 3

Extract:

* Wallet Service
* Ranking Service

---

Phase 4

Multi Region Deployment

```
```
