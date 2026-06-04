package redis

import (
	"fmt"

	"github.com/google/uuid"
)

// MatchStateKey returns the Redis key for match state.
// Format: "match:{matchId}:state"
// TTL: 2 hours
func MatchStateKey(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s:state", matchID)
}

// UserConnectionKey returns the Redis key for user connection state.
// Format: "user:{userId}:connection"
// TTL: 60 seconds
func UserConnectionKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:connection", userID)
}

// QueueKey returns the Redis key for a matchmaking queue.
// Format: "queue:{region}:{difficulty}:{stake}"
// Type: Sorted Set, score = join timestamp
// Per DEC-015: 3-segment key (region, difficulty, stake).
func QueueKey(region, difficulty, stake string) string {
	return fmt.Sprintf("queue:%s:%s:%s", region, difficulty, stake)
}

// RateMoveKey returns the Redis key for rate-limiting user moves.
// Format: "rate:user:{userId}:move"
// TTL: 1 second (per DEC-011: 5 moves/sec limit)
func RateMoveKey(userID uuid.UUID) string {
	return fmt.Sprintf("rate:user:%s:move", userID)
}

// ReconnectKey returns the Redis key for reconnect state.
// Format: "match:{matchId}:reconnect:{userId}"
// TTL: 30 seconds (per DEC-010)
func ReconnectKey(matchID, userID uuid.UUID) string {
	return fmt.Sprintf("match:%s:reconnect:%s", matchID, userID)
}
