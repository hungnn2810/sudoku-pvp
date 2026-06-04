CREATE TABLE user_providers (
    id               UUID         PRIMARY KEY,
    user_id          UUID         NOT NULL REFERENCES users(id),
    provider         VARCHAR(50)  NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    email            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_provider_user UNIQUE (provider, provider_user_id)
);
CREATE INDEX idx_user_providers_user ON user_providers(user_id);
