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
