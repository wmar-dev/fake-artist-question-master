package game

import (
	"strings"
	"sync"
)

const (
	codeLength = 5
	// codeAlphabet leaves out the look-alike characters 0, O, 1, I, and L.
	codeAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"
)

// Store holds all live games. Its lock is never held while taking the lock of
// a game that is already in the store.
type Store struct {
	clock    Clock
	rng      Rand
	words    []WordEntry
	maxGames int

	mu    sync.Mutex
	games map[string]*Game
}

// NewStore returns an empty Store that allows up to maxGames live games.
func NewStore(clock Clock, rng Rand, words []WordEntry, maxGames int) *Store {
	return &Store{clock: clock, rng: rng, words: words, maxGames: maxGames, games: map[string]*Game{}}
}

// Create starts a game and makes hostName its host.
func (s *Store) Create(hostName string) (*Game, Identity, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.games) >= s.maxGames {
		return nil, Identity{}, ErrServerFull
	}
	code := s.newCode()
	g, host, err := newGame(code, hostName, s.clock, s.rng, s.words, s.remove)
	if err != nil {
		return nil, Identity{}, err
	}
	s.games[code] = g
	return g, host, nil
}

// Get finds a live game by its code, ignoring case.
func (s *Store) Get(code string) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.games[strings.ToUpper(code)]
	if !ok {
		return nil, ErrGameNotFound
	}
	return g, nil
}

// Count returns the number of live games.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.games)
}

// CloseAll ends every game, telling connected players why.
func (s *Store) CloseAll(reason string) {
	s.mu.Lock()
	games := s.games
	s.games = map[string]*Game{}
	s.mu.Unlock()
	for _, g := range games {
		g.close(reason)
	}
}

func (s *Store) remove(g *Game) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, g.code)
}

// newCode returns a code that no live game is using.
func (s *Store) newCode() string {
	for {
		code := make([]byte, codeLength)
		for i := range code {
			code[i] = codeAlphabet[s.rng.IntN(len(codeAlphabet))]
		}
		if _, taken := s.games[string(code)]; !taken {
			return string(code)
		}
	}
}
