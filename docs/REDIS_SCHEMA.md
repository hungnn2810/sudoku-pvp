# REDIS_SCHEMA.md

## Key Naming Convention

module:resource:id

Examples:

match:{matchId}:state
user:{userId}:connection
queue:{region}:{difficulty}:{stake}

---

## Match State

Key:
match:{matchId}:state

TTL:
2 hours

Stores:
- score
- combo
- progress
- status
- timer

---

## Online User

Key:
user:{userId}:connection

TTL:
60 seconds

Stores:
- connection id
- last heartbeat

---

## Matchmaking Queue

queue:{region}:{difficulty}:{stake}

Type:
Sorted Set

Score:
join timestamp

---

## Rate Limit

rate:user:{userId}:move

TTL:
1 second

Stores:
move count

---

## Reconnect State

match:{matchId}:reconnect:{userId}

TTL:
30 seconds

---

## Refresh Token

Key:
refresh:{userId}

TTL:
30 days (720 hours)

Stores:
- opaque refresh token string (single token per user, per D-07)
- overwritten on each new login and each token rotation (D-01, D-03)
