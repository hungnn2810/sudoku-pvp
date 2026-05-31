# Domain Model

## Overview

Sudoku Battle Arena được thiết kế theo Domain Driven Design (DDD).

Core Domain:

* Battle
* Matchmaking
* Ranking
* Wallet

Supporting Domain:

* User
* Mission
* Shop
* Analytics

---

# Aggregate: User

## User

```go
type User struct {
    ID          uuid.UUID
    Username    string
    AvatarURL   string

    Level       int
    Exp         int

    RankTier    RankTier
    RankPoint   int

    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

Rules:

* Username unique
* RankPoint >= 0
* Level >= 1

---

# Aggregate: Wallet

## Wallet

```go
type Wallet struct {
    UserID      uuid.UUID

    CoinBalance int64
    GemBalance  int64
}
```

Invariants:

```text
CoinBalance >= 0
GemBalance >= 0
```

---

## WalletTransaction

```go
type WalletTransaction struct {
    ID              uuid.UUID

    UserID          uuid.UUID

    Type            TransactionType

    Amount          int64

    BalanceBefore   int64
    BalanceAfter    int64

    ReferenceType   string
    ReferenceID     string

    CreatedAt       time.Time
}
```

---

# Aggregate: Sudoku Puzzle

## SudokuPuzzle

```go
type SudokuPuzzle struct {
    ID              uuid.UUID

    Difficulty      Difficulty

    PuzzleGrid      string
    SolutionGrid    string

    EmptyCount      int

    CreatedAt       time.Time
}
```

Rules:

* SolutionGrid never exposed to client
* Puzzle immutable after creation

---

# Aggregate: Match

## Match

```go
type Match struct {
    ID              uuid.UUID

    Mode            MatchMode

    Difficulty      Difficulty

    StakeCoin       int64

    Status          MatchStatus

    PuzzleID        uuid.UUID

    StartedAt       *time.Time
    EndedAt         *time.Time
}
```

---

## MatchPlayer

```go
type MatchPlayer struct {
    MatchID         uuid.UUID
    UserID          uuid.UUID

    Score           int

    Progress        int

    Combo           int

    WrongCount      int

    HintUsed        int

    Result          MatchResult

    CoinChange      int64
    RankChange      int

    FinishedAt      *time.Time
}
```

---

## MatchMove

```go
type MatchMove struct {
    ID              uuid.UUID

    MatchID         uuid.UUID
    UserID          uuid.UUID

    Row             int
    Col             int

    Value           int

    IsCorrect       bool

    ScoreDelta      int

    CreatedAt       time.Time
}
```

---

# Aggregate: Ranking

## RankTier

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

## RankProfile

```go
type RankProfile struct {
    UserID      uuid.UUID

    Tier        RankTier
    Point       int

    WinCount    int
    LoseCount   int
}
```

---

# Aggregate: Mission

## Mission

```go
type Mission struct {
    ID              uuid.UUID

    Name            string

    Type            MissionType

    TargetValue     int

    RewardCoin      int64

    RewardExp       int

    Active          bool
}
```

---

## UserMission

```go
type UserMission struct {
    UserID      uuid.UUID
    MissionID   uuid.UUID

    Progress    int

    Claimed     bool
}
```

---

# Aggregate: Shop

## ShopItem

```go
type ShopItem struct {
    ID              uuid.UUID

    Type            ItemType

    Name            string

    CoinPrice       int64
    GemPrice        int64

    Metadata        json.RawMessage

    Active          bool
}
```

---

## InventoryItem

```go
type InventoryItem struct {
    UserID      uuid.UUID

    ItemID      uuid.UUID

    Equipped    bool
}
```
