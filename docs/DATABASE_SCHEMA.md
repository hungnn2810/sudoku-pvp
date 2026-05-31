# Database Schema

Database: PostgreSQL

Version: 1.0

---

# users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,

    username VARCHAR(50) NOT NULL UNIQUE,

    avatar_url TEXT,

    level INT NOT NULL DEFAULT 1,

    exp INT NOT NULL DEFAULT 0,

    rank_tier VARCHAR(20) NOT NULL DEFAULT 'Bronze',

    rank_point INT NOT NULL DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);
```

---

# wallets

```sql
CREATE TABLE wallets (
    user_id UUID PRIMARY KEY REFERENCES users(id),

    coin_balance BIGINT NOT NULL DEFAULT 0,

    gem_balance BIGINT NOT NULL DEFAULT 0,

    updated_at TIMESTAMPTZ NOT NULL
);
```

---

# wallet_transactions

```sql
CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES users(id),

    type VARCHAR(50) NOT NULL,

    amount BIGINT NOT NULL,

    balance_before BIGINT NOT NULL,

    balance_after BIGINT NOT NULL,

    reference_type VARCHAR(50),

    reference_id VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL
);
```

---

# sudoku_puzzles

```sql
CREATE TABLE sudoku_puzzles (
    id UUID PRIMARY KEY,

    difficulty VARCHAR(20) NOT NULL,

    puzzle_grid JSONB NOT NULL,

    solution_grid JSONB NOT NULL,

    empty_count INT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL
);
```

---

# matches

```sql
CREATE TABLE matches (
    id UUID PRIMARY KEY,

    mode VARCHAR(20) NOT NULL,

    status VARCHAR(20) NOT NULL,

    difficulty VARCHAR(20) NOT NULL,

    puzzle_id UUID NOT NULL REFERENCES sudoku_puzzles(id),

    stake_coin BIGINT NOT NULL,

    started_at TIMESTAMPTZ,

    ended_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL
);
```

---

# match_players

```sql
CREATE TABLE match_players (
    id UUID PRIMARY KEY,

    match_id UUID NOT NULL REFERENCES matches(id),

    user_id UUID NOT NULL REFERENCES users(id),

    score INT NOT NULL DEFAULT 0,

    progress INT NOT NULL DEFAULT 0,

    combo INT NOT NULL DEFAULT 0,

    wrong_count INT NOT NULL DEFAULT 0,

    hint_used INT NOT NULL DEFAULT 0,

    result VARCHAR(20),

    coin_change BIGINT DEFAULT 0,

    rank_change INT DEFAULT 0,

    finished_at TIMESTAMPTZ
);
```

---

# match_moves

```sql
CREATE TABLE match_moves (
    id UUID PRIMARY KEY,

    match_id UUID NOT NULL REFERENCES matches(id),

    user_id UUID NOT NULL REFERENCES users(id),

    row_index SMALLINT NOT NULL,

    col_index SMALLINT NOT NULL,

    value SMALLINT NOT NULL,

    is_correct BOOLEAN NOT NULL,

    score_delta INT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL
);
```

---

# missions

```sql
CREATE TABLE missions (
    id UUID PRIMARY KEY,

    type VARCHAR(20) NOT NULL,

    name VARCHAR(100) NOT NULL,

    description TEXT,

    target_value INT NOT NULL,

    reward_coin BIGINT NOT NULL,

    reward_exp INT NOT NULL,

    is_active BOOLEAN NOT NULL
);
```

---

# user_missions

```sql
CREATE TABLE user_missions (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES users(id),

    mission_id UUID NOT NULL REFERENCES missions(id),

    progress INT NOT NULL DEFAULT 0,

    claimed BOOLEAN NOT NULL DEFAULT FALSE
);
```

---

# shop_items

```sql
CREATE TABLE shop_items (
    id UUID PRIMARY KEY,

    type VARCHAR(50) NOT NULL,

    name VARCHAR(100) NOT NULL,

    coin_price BIGINT,

    gem_price BIGINT,

    metadata JSONB,

    is_active BOOLEAN NOT NULL
);
```

---

# inventory_items

```sql
CREATE TABLE inventory_items (
    id UUID PRIMARY KEY,

    user_id UUID NOT NULL REFERENCES users(id),

    item_id UUID NOT NULL REFERENCES shop_items(id),

    equipped BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL
);
```

---

# Recommended Indexes

```sql
CREATE INDEX idx_match_players_user
ON match_players(user_id);

CREATE INDEX idx_match_moves_match
ON match_moves(match_id);

CREATE INDEX idx_wallet_transactions_user
ON wallet_transactions(user_id);

CREATE INDEX idx_user_rank
ON users(rank_tier, rank_point DESC);
```
