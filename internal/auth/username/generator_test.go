package username_test

import (
	"regexp"
	"testing"

	"sudoku-pvp/internal/auth/username"
)

func TestGenerate_Format(t *testing.T) {
	// Pattern: capital letter + lowercase letters (adjective) +
	//          capital letter + lowercase letters (noun) +
	//          two digits (number 10-99).
	pattern := regexp.MustCompile(`^[A-Z][a-z]+[A-Z][a-z]+\d{2}$`)

	for i := 0; i < 100; i++ {
		result := username.Generate()
		if !pattern.MatchString(result) {
			t.Errorf("Generate() = %q does not match pattern %s", result, pattern)
		}
	}
}

func TestGenerate_Length(t *testing.T) {
	result := username.Generate()
	// Shortest possible: 3-char adj + 3-char noun + 2 digits = 8 chars
	// Longest possible: longest adj (5 chars) + longest noun (7 chars) + 2 digits = 14 chars
	// We use generous bounds to accommodate any word list changes.
	if len(result) < 6 {
		t.Errorf("Generate() = %q has length %d, want >= 6", result, len(result))
	}
	if len(result) > 25 {
		t.Errorf("Generate() = %q has length %d, want <= 25", result, len(result))
	}
}

func TestGenerate_Uniqueness(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		seen[username.Generate()] = struct{}{}
	}
	// Require at least 50% unique names (> 500 out of 1000).
	if len(seen) <= 500 {
		t.Errorf("Generate() only produced %d unique names out of %d — distribution too narrow", len(seen), n)
	}
}
