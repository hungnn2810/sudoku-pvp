CREATE TABLE user_missions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    mission_id UUID NOT NULL REFERENCES missions(id),
    progress INT NOT NULL DEFAULT 0,
    claimed BOOLEAN NOT NULL DEFAULT FALSE
);
