CREATE TABLE users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    avatar_url TEXT,
    level INT NOT NULL DEFAULT 1,
    exp INT NOT NULL DEFAULT 0,
    rank_tier VARCHAR(20) NOT NULL DEFAULT 'Bronze',
    rank_point INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
