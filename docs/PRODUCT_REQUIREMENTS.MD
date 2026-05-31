# Product Requirements Document (PRD)

Project: Sudoku Battle Arena

Version: MVP 1.0

---

# Product Vision

Sudoku Battle Arena là game Sudoku cạnh tranh thời gian thực.

Người chơi có thể:

* Chơi Solo
* Chơi với Bot
* Chơi PvP Realtime
* Leo Rank
* Hoàn thành Daily Mission
* Thu thập Cosmetic

Mục tiêu:

* Session ngắn
* Gameplay gây nghiện
* PvP căng thẳng
* Không pay-to-win

---

# User Personas

## Casual Player

Mục tiêu:

* Giải Sudoku
* Giết thời gian

---

## Competitive Player

Mục tiêu:

* Leo rank
* So tài với người chơi khác

---

## Collector

Mục tiêu:

* Mở khóa skin
* Thu thập avatar
* Hoàn thành achievement

---

# MVP Scope

## Included

### Authentication

* Guest Login
* Google Login

### Solo

* Easy
* Medium
* Hard

### Bot Battle

* Easy Bot
* Medium Bot
* Hard Bot

### PvP

* Matchmaking
* Room
* Battle
* Coin Stake

### Economy

* Coin

### Profile

* Statistics
* Match History

---

## Excluded

* Clan
* Tournament
* Spectator
* Replay
* Battle Pass
* Guild
* Chat System

---

# Core Metrics

## D1 Retention

Target:

```text
>= 35%
```

## D7 Retention

Target:

```text
>= 15%
```

## Match Completion Rate

Target:

```text
>= 90%
```

## Average Match Duration

Target:

```text
3 - 7 minutes
```

## Matchmaking Time

Target:

```text
< 10 seconds
```

---

# Functional Requirements

## Solo

Player can:

* Start game
* Pause game
* Resume game
* Use hint
* Finish game

---

## PvP

Player can:

* Join queue
* Match opponent
* Ready
* Play
* Reconnect
* Surrender

---

## Wallet

Player can:

* Earn coin
* Spend coin
* View transactions

---

## Ranking

Player can:

* Gain rank point
* Lose rank point
* View leaderboard

---

# Non Functional Requirements

Availability

```text
99.9%
```

Battle Validation

```text
<20ms
```

WebSocket Latency

```text
<100ms
```

Matchmaking

```text
<10 seconds
```
