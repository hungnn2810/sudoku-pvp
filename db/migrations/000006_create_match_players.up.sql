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
