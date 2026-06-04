package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

// AuthRepo manages refresh token storage in Redis.
// D-01: key pattern "refresh:{userId}", single token per user.
// D-02: 30-day TTL matches Redis key TTL.
type AuthRepo struct {
	rdb *goredis.Client
}

// NewAuthRepo creates a new AuthRepo backed by the given Redis client.
func NewAuthRepo(rdb *goredis.Client) *AuthRepo {
	return &AuthRepo{rdb: rdb}
}

// refreshKey returns the Redis key for a user's refresh token.
// Format: "refresh:{userId}"
// TTL: 30 days (per D-01, D-02)
func refreshKey(userID uuid.UUID) string {
	return "refresh:" + userID.String()
}

// StoreRefreshToken saves token under "refresh:{userId}" with the given TTL.
// D-01: new login overwrites previous token (single token per user).
// D-03: called after rotation — old key deleted, new key set atomically via SET.
func (r *AuthRepo) StoreRefreshToken(ctx context.Context, userID uuid.UUID, token string, ttl time.Duration) error {
	if err := r.rdb.Set(ctx, refreshKey(userID), token, ttl).Err(); err != nil {
		return fmt.Errorf("auth repo store refresh token: %w", err)
	}
	return nil
}

// GetRefreshToken retrieves the stored refresh token for the given user.
// Returns a wrapped redis.Nil error if the key does not exist (token expired or never set).
func (r *AuthRepo) GetRefreshToken(ctx context.Context, userID uuid.UUID) (string, error) {
	val, err := r.rdb.Get(ctx, refreshKey(userID)).Result()
	if err == goredis.Nil {
		return "", fmt.Errorf("auth repo get refresh token: %w", goredis.Nil)
	}
	if err != nil {
		return "", fmt.Errorf("auth repo get refresh token: %w", err)
	}
	return val, nil
}

// DeleteRefreshToken removes the refresh token for the given user from Redis.
// D-06: idempotent — if key does not exist (already deleted or never set), returns nil.
func (r *AuthRepo) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	if err := r.rdb.Del(ctx, refreshKey(userID)).Err(); err != nil {
		return fmt.Errorf("auth repo delete refresh token: %w", err)
	}
	return nil
}
