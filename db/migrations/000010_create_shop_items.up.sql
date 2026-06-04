CREATE TABLE shop_items (
    id UUID PRIMARY KEY,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    coin_price BIGINT,
    gem_price BIGINT,
    metadata JSONB,
    is_active BOOLEAN NOT NULL
);
