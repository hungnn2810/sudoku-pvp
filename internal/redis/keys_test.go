package redis

import (
	"testing"

	"github.com/google/uuid"
)

// Fixed UUID for deterministic assertions.
const fixedUUIDStr = "018f2345-6789-7abc-def0-1234567890ab"

func fixedUUID() uuid.UUID {
	id, err := uuid.Parse(fixedUUIDStr)
	if err != nil {
		panic("fixedUUID: invalid UUID constant: " + err.Error())
	}
	return id
}

func TestMatchStateKey(t *testing.T) {
	id := fixedUUID()
	got := MatchStateKey(id)
	want := "match:" + id.String() + ":state"
	if got != want {
		t.Errorf("MatchStateKey(%s) = %q; want %q", id, got, want)
	}
}

func TestUserConnectionKey(t *testing.T) {
	id := fixedUUID()
	got := UserConnectionKey(id)
	want := "user:" + id.String() + ":connection"
	if got != want {
		t.Errorf("UserConnectionKey(%s) = %q; want %q", id, got, want)
	}
}

func TestQueueKey(t *testing.T) {
	got := QueueKey("asia", "medium", "100")
	want := "queue:asia:medium:100"
	if got != want {
		t.Errorf("QueueKey(asia, medium, 100) = %q; want %q", got, want)
	}
}

func TestRateMoveKey(t *testing.T) {
	id := fixedUUID()
	got := RateMoveKey(id)
	want := "rate:user:" + id.String() + ":move"
	if got != want {
		t.Errorf("RateMoveKey(%s) = %q; want %q", id, got, want)
	}
}

func TestReconnectKey(t *testing.T) {
	matchID := fixedUUID()
	userID, _ := uuid.Parse("019a1234-5678-8def-abcd-ef0123456789")
	got := ReconnectKey(matchID, userID)
	want := "match:" + matchID.String() + ":reconnect:" + userID.String()
	if got != want {
		t.Errorf("ReconnectKey(%s, %s) = %q; want %q", matchID, userID, got, want)
	}
}

func TestKeyFormat_NoRawStrings(t *testing.T) {
	// Verify all functions use dynamic UUID formatting (not hardcoded strings).
	id1 := fixedUUID()
	id2, _ := uuid.Parse("019a1234-5678-8def-abcd-ef0123456789")

	// Each function must produce different output for different IDs.
	if MatchStateKey(id1) == MatchStateKey(id2) {
		t.Error("MatchStateKey: same output for different UUIDs — likely hardcoded")
	}
	if UserConnectionKey(id1) == UserConnectionKey(id2) {
		t.Error("UserConnectionKey: same output for different UUIDs — likely hardcoded")
	}
	if RateMoveKey(id1) == RateMoveKey(id2) {
		t.Error("RateMoveKey: same output for different UUIDs — likely hardcoded")
	}
	if ReconnectKey(id1, id2) == ReconnectKey(id2, id1) {
		t.Error("ReconnectKey: same output for swapped UUIDs — likely hardcoded")
	}
}
