# API Contract

Base URL

```text
/api/v1
```

Authentication

```http
Authorization: Bearer {jwt}
```

---

# Auth

## Guest Login

POST /auth/guest

Response

```json
{
  "accessToken":"...",
  "refreshToken":"..."
}
```

---

# User

## Get Profile

GET /me

Response

```json
{
  "id":"uuid",
  "username":"Player001",
  "avatarUrl":"",
  "level":10,
  "rank":"Silver II",
  "coin":500
}
```

---

# Wallet

GET /wallet

Response

```json
{
  "coin":5000,
  "gem":50
}
```

---

GET /wallet/transactions

Response

```json
{
  "items":[]
}
```

---

# Sudoku

GET /sudoku/puzzle?difficulty=medium

Response

```json
{
  "id":"puzzle_1",
  "difficulty":"medium",
  "grid":[]
}
```

---

# Match History

GET /matches

Response

```json
{
  "items":[]
}
```

---

GET /matches/{id}

Response

```json
{
  "id":"match_1",
  "players":[]
}
```

---

# Ranking

GET /ranking/me

Response

```json
{
  "tier":"Silver",
  "point":1500
}
```

---

GET /ranking/leaderboard

Response

```json
{
  "items":[]
}
```

---

# Mission

GET /missions

Response

```json
{
  "items":[]
}
```

---

POST /missions/{id}/claim

Response

```json
{
  "coinReward":100
}
```

---

# Shop

GET /shop/items

Response

```json
{
  "items":[]
}
```

---

POST /shop/buy

Request

```json
{
  "itemId":"uuid"
}
```

Response

```json
{
  "success":true
}
```

---

# WebSocket

## Connect

```http
GET /ws/connect
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

battle.reconnect_state
```
