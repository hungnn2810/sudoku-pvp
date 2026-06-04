package username

import (
	"fmt"
	"math/rand/v2"
)

// adjectives and nouns are embedded word lists (D-15: no external library).
// At least 20 entries each per plan requirement.
var adjectives = []string{
	"Swift", "Bold", "Calm", "Dark", "Epic",
	"Iron", "Jade", "Keen", "Lone", "Mist",
	"Nova", "Onyx", "Pure", "Rune", "Sage",
	"Tide", "Umber", "Void", "Wild", "Zen",
	"Fern", "Gray", "Haze", "Icy", "Lush",
}

var nouns = []string{
	"King", "Wolf", "Fox", "Bear", "Hawk",
	"Crow", "Dragon", "Eagle", "Falcon", "Ghost",
	"Hunter", "Knight", "Lion", "Monk", "Ninja",
	"Oracle", "Panda", "Quest", "Rider", "Sage",
	"Tiger", "Viper", "Wizard", "Yeti", "Zen",
}

// Generate returns a random "AdjectiveNounNumber" username.
// D-15: format e.g. "SwiftKing42". Number range 10-99 for brevity.
// Collision retry is handled by the caller (UserRepo.CreateGuestUser).
func Generate() string {
	adj := adjectives[rand.IntN(len(adjectives))]
	noun := nouns[rand.IntN(len(nouns))]
	num := 10 + rand.IntN(90)
	return fmt.Sprintf("%s%s%d", adj, noun, num)
}
