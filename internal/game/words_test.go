package game_test

import (
	"strings"
	"testing"

	"fakeartist/internal/game"
)

func TestWordListSize(t *testing.T) {
	words := game.Words()
	if len(words) < 200 {
		t.Errorf("got %d words, want at least 200", len(words))
	}
	categories := map[string]bool{}
	for _, w := range words {
		categories[w.Category] = true
	}
	if len(categories) < 10 {
		t.Errorf("got %d categories, want at least 10", len(categories))
	}
}

func TestWordListHasNoDuplicatesOrBlanks(t *testing.T) {
	seen := map[string]bool{}
	for _, w := range game.Words() {
		if w.Category == "" || w.Word == "" {
			t.Errorf("blank field in %+v", w)
		}
		key := strings.ToLower(w.Word)
		if seen[key] {
			t.Errorf("duplicate word %q", w.Word)
		}
		seen[key] = true
	}
}
