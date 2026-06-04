package username

import (
	"fmt"
	"math/rand/v2"
)

// adjectives is the embedded word list for the first component of guest usernames.
// D-15: no external library — words are embedded directly.
var adjectives = []string{
	"Swift", "Bold", "Calm", "Dark", "Epic",
	"Fierce", "Gloom", "Hazy", "Iron", "Jade",
	"Keen", "Lofty", "Mist", "Noble", "Onyx",
	"Prime", "Quick", "Royal", "Storm", "Tidal",
	"Ultra", "Vivid", "Wild", "Xeno", "Young", "Zesty",
}

// nouns is the embedded word list for the second component of guest usernames.
// D-15: no external library — words are embedded directly.
var nouns = []string{
	"King", "Wolf", "Fox", "Bear", "Hawk",
	"Lion", "Sage", "Monk", "Drake", "Blade",
	"Crest", "Dawn", "Edge", "Flame", "Ghost",
	"Heart", "Isle", "Jade", "Knight", "Lance",
	"Mark", "Nord", "Oak", "Peak", "Quest",
	"Reef", "Star", "Tide", "Vale", "Warden",
}

// Generate returns a random "AdjectiveNounNumber" username.
// D-15: format e.g. "SwiftKing42". Number range 10-99 for brevity.
// This function is pure — no I/O, collision detection is handled by the caller.
func Generate() string {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	num := 10 + rand.IntN(90)
	return fmt.Sprintf("%s%s%d", adj, noun, num)
}
