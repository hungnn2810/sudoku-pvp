CREATE TABLE wallets (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    coin_balance BIGINT NOT NULL DEFAULT 0,
    gem_balance BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL
);
