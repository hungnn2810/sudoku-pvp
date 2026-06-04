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
