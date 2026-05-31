# Event Catalog

Version: 1.0

Message Broker: RabbitMQ

Exchange:

```text
game.events
```

---

# MatchStarted

Publisher

```text
Battle Module
```

Routing Key

```text
match.started
```

Payload

```json
{
  "matchId":"uuid",
  "mode":"pvp",
  "difficulty":"medium"
}
```

---

# MatchFinished

Routing Key

```text
match.finished
```

Payload

```json
{
  "matchId":"uuid",
  "winnerId":"uuid",
  "duration":320
}
```

Consumers

* Ranking
* Wallet
* Mission
* Analytics

---

# WalletTransactionCreated

Routing Key

```text
wallet.transaction.created
```

Payload

```json
{
  "transactionId":"uuid",
  "userId":"uuid",
  "amount":100
}
```

---

# RankChanged

Routing Key

```text
ranking.changed
```

Payload

```json
{
  "userId":"uuid",
  "oldRank":"Silver",
  "newRank":"Gold"
}
```

---

# MissionCompleted

Routing Key

```text
mission.completed
```

Payload

```json
{
  "userId":"uuid",
  "missionId":"uuid"
}
```

---

# ShopItemPurchased

Routing Key

```text
shop.purchased
```

Payload

```json
{
  "userId":"uuid",
  "itemId":"uuid"
}
```

---

# WebSocket Events

## Client → Server

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

## Server → Client

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

battle.error
```
