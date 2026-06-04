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
