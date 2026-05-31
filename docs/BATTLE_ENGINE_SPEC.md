# BATTLE_ENGINE_SPEC.md

Version: 1.0

Owner: Backend Team

Status: Approved

---

# Purpose

This document defines the authoritative battle engine specification for Sudoku Battle Arena.

The Battle Engine is responsible for:

* Move validation
* Scoring
* Combo system
* Match progression
* Win/Loss calculation
* Realtime synchronization
* Reconnect handling
* Anti-cheat enforcement

The Battle Engine is the most critical backend component.

---

# Design Principles

## Server Authoritative

Client is never trusted.

Client cannot determine:

* Score
* Combo
* Progress
* Rewards
* Rank changes
* Match result

All calculations must happen on server.

---

## Deterministic

The same input must always produce the same output.

Example:

```text
Move:
Row=3
Col=5
Value=7

Result:
Always identical
```

No randomness allowed during battle.

---

## Stateless API

Battle state must be loaded from Redis.

Never rely on in-memory state.

This allows:

* Horizontal scaling
* Reconnect support
* Pod restart recovery

---

# Match Lifecycle

## State Machine

```text
CREATED

↓

WAITING_PLAYERS

↓

READY

↓

COUNTDOWN

↓

PLAYING

↓

FINAL_COUNTDOWN

↓

FINISHED
```

---

# State Definitions

## CREATED

Room exists.

Players not joined yet.

---

## WAITING_PLAYERS

Players entering room.

Waiting ready status.

---

## READY

Both players ready.

---

## COUNTDOWN

Countdown begins.

Duration:

```text
3 seconds
```

---

## PLAYING

Main gameplay state.

Accepts:

```text
battle.submit_move

battle.use_hint

battle.surrender
```

---

## FINAL_COUNTDOWN

Triggered when one player completes puzzle.

Duration:

```text
30 seconds
```

Opponent may continue playing.

---

## FINISHED

Match closed.

Rewards calculated.

Result persisted.

No more moves accepted.

---

# Match Configuration

## Easy

```text
Duration = 300 seconds
```

---

## Medium

```text
Duration = 420 seconds
```

---

## Hard

```text
Duration = 600 seconds
```

---

## Expert

```text
Duration = 720 seconds
```

---

# Battle State

Redis Key

```text
match:{matchId}:state
```

Structure

```json
{
  "matchId":"uuid",
  "status":"playing",
  "startedAt":"2026-06-01T10:00:00Z",
  "remainingSeconds":420,

  "players":{
    "user1":{
      "score":120,
      "progress":45,
      "combo":3,
      "wrongCount":1,
      "finished":false
    },

    "user2":{
      "score":90,
      "progress":40,
      "combo":2,
      "wrongCount":2,
      "finished":false
    }
  }
}
```

---

# Move Submission

Client Event

```json
{
  "event":"battle.submit_move",
  "data":{
    "matchId":"uuid",
    "row":3,
    "col":5,
    "value":7
  }
}
```

---

# Validation Pipeline

## Step 1

Match exists.

---

## Step 2

Match status = PLAYING

---

## Step 3

User belongs to match.

---

## Step 4

Cell coordinates valid.

```text
0 <= row <= 8

0 <= col <= 8
```

---

## Step 5

Cell is editable.

Cannot modify puzzle fixed cells.

---

## Step 6

Cell not already completed.

---

## Step 7

Value valid.

```text
1 <= value <= 9
```

---

## Step 8

Compare with solution.

---

## Step 9

Calculate score.

---

## Step 10

Update combo.

---

## Step 11

Update progress.

---

## Step 12

Check completion rules.

---

## Step 13

Broadcast result.

---

# Scoring System

## Correct Move

```text
+10
```

---

## Wrong Move

```text
-5
```

---

## Complete Row

```text
+20
```

---

## Complete Column

```text
+20
```

---

## Complete Box

```text
+15
```

---

## Puzzle Completion

```text
+100
```

---

# Combo System

## Combo Definition

Consecutive correct moves.

Wrong move resets combo.

---

## Combo 1

```text
Multiplier = 1.0
```

---

## Combo 2

```text
Multiplier = 1.2
```

---

## Combo 3+

```text
Multiplier = 1.4
```

---

## Example

```text
Correct
Correct
Correct

Base Score:

10 + 10 + 10

Combo:

+4

Total:

34
```

---

# Wrong Move Penalty

Wrong move triggers:

```text
Score -5

Combo Reset

Input Lock
```

---

## Input Lock

Duration:

```text
2 seconds
```

Player cannot submit move.

Server validates lock.

Client lock animation is cosmetic only.

---

# Progress Calculation

Progress:

```text
filled_cells / total_empty_cells
```

Example:

```text
45 / 60

Progress = 75%
```

---

# Row Completion

Award once.

Conditions:

```text
All cells filled

All values correct
```

Reward:

```text
+20
```

---

# Column Completion

Award once.

Reward:

```text
+20
```

---

# Box Completion

3x3 region.

Reward:

```text
+15
```

---

# Puzzle Completion

Conditions:

```text
All empty cells filled

All correct
```

Actions:

```text
Mark Finished

Award 100 points

Trigger Final Countdown
```

---

# Final Countdown

Triggered when:

```text
First player finishes puzzle
```

Event:

```json
{
  "event":"battle.final_countdown",
  "data":{
    "remainingSeconds":30
  }
}
```

---

# Winner Determination

## Rule 1

Higher score wins.

---

## Rule 2

If score equal:

Earlier finish time wins.

---

## Rule 3

If still equal:

Draw.

---

# Surrender

Client Event

```json
{
  "event":"battle.surrender"
}
```

Result:

```text
Player loses immediately
```

Effects:

```text
Rank Loss

Coin Loss

Match End
```

---

# Timeout

Condition:

```text
Remaining time = 0
```

Winner:

```text
Highest score
```

---

# Disconnect Handling

## Grace Period

```text
30 seconds
```

---

## During Disconnect

Match continues.

Opponent continues playing.

---

## Reconnect Success

Server sends:

```json
{
  "event":"battle.reconnect_state",
  "data":{
    "score":120,
    "combo":3,
    "progress":45,
    "remainingSeconds":120
  }
}
```

---

## Reconnect Failed

After 30 seconds:

```text
Automatic Surrender
```

---

# Anti Cheat Rules

## Rule 1

Client cannot send score.

Reject request.

---

## Rule 2

Client cannot send combo.

Reject request.

---

## Rule 3

Client cannot send progress.

Reject request.

---

## Rule 4

Client cannot send result.

Reject request.

---

## Rule 5

Client never receives solution.

Only puzzle grid.

---

# Rate Limiting

## Move Limit

```text
5 moves / second
```

---

## First Violation

```text
Warning
```

---

## Second Violation

```text
Temporary Input Lock
```

---

## Third Violation

```text
Match Forfeit
```

---

# Event Emission

## Move Accepted

Publish:

```text
battle.move.accepted
```

---

## Move Rejected

Publish:

```text
battle.move.rejected
```

---

## Match Finished

Publish:

```text
match.finished
```

Consumers:

* Wallet
* Ranking
* Mission
* Analytics

---

# Persistence Strategy

Realtime State

```text
Redis
```

---

Historical Data

```text
PostgreSQL
```

---

# Performance Requirements

Move Validation

```text
P95 < 20ms
```

---

Broadcast

```text
P95 < 100ms
```

---

Reconnect Recovery

```text
< 1 second
```

---

# Future Extensions

Reserved Features

* Tournament Mode
* Spectator Mode
* Replay System
* Power Ups
* Team Battle
* Clan Battle

Current Battle Engine must remain compatible with future extensions.
