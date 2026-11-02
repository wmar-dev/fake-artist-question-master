package game

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed words.txt
var wordData string

// WordEntry is a category and word that a round can use.
type WordEntry struct {
	Category string
	Word     string
}

// Words returns the built-in word list.
var Words = sync.OnceValue(func() []WordEntry {
	var words []WordEntry
	for line := range strings.Lines(wordData) {
		category, word, ok := strings.Cut(strings.TrimSpace(line), ",")
		if ok {
			words = append(words, WordEntry{Category: category, Word: word})
		}
	}
	return words
})
