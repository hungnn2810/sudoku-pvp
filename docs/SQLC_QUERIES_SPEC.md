# SQLC_QUERIES_SPEC

## Principles
- One query per use case
- No SELECT *
- Explicit columns
- Generated code wrapped by repository layer

## Users
-- name: GetUserByID :one
SELECT id, username, avatar_url, level, exp, rank_tier, rank_point
FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (...) VALUES (...) RETURNING *;

## Wallet
-- name: GetWalletByUserID :one
SELECT user_id, coin_balance, gem_balance
FROM wallets
WHERE user_id = $1;

-- name: CreateWalletTransaction :one
INSERT INTO wallet_transactions (...)
VALUES (...)
RETURNING *;

## Match
-- name: CreateMatch :one
INSERT INTO matches (...)
VALUES (...)
RETURNING *;

-- name: GetMatchByID :one
SELECT * FROM matches WHERE id=$1;

## Ranking
-- name: GetLeaderboard :many
SELECT id, username, rank_tier, rank_point
FROM users
ORDER BY rank_point DESC
LIMIT $1;
