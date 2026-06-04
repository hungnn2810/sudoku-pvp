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
