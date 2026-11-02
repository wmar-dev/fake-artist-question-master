package game

// Round is one deal of the game. Names are copied so that the result can still
// be shown after those players leave.
type round struct {
	number       int
	category     string
	word         string
	fakeID       string
	fakeName     string
	firstName    string
	participants map[string]bool
	revealed     bool
}

// shuffledDeck returns the indexes of n words in random order. A game steps
// through its deck, so words come back oldest first once it is used up.
func shuffledDeck(n int, rng Rand) []int {
	deck := make([]int, n)
	for i := range deck {
		deck[i] = i
	}
	for i := n - 1; i > 0; i-- {
		j := rng.IntN(i + 1)
		deck[i], deck[j] = deck[j], deck[i]
	}
	return deck
}

// Deal starts a new round, discarding any current one.
func (g *Game) Deal(byID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrGameNotFound
	}
	if byID != g.hostID {
		return ErrNotHost
	}
	if len(g.players) < MinPlayers {
		return ErrWrongPlayerCount
	}

	entry := g.words[g.deck[g.nextWord]]
	g.nextWord = (g.nextWord + 1) % len(g.deck)
	fake := g.players[g.rng.IntN(len(g.players))]
	first := g.players[g.rng.IntN(len(g.players))]

	participants := make(map[string]bool, len(g.players))
	for _, p := range g.players {
		participants[p.id] = true
	}
	number := 1
	if g.round != nil {
		number = g.round.number + 1
	}
	g.round = &round{
		number:       number,
		category:     entry.Category,
		word:         entry.Word,
		fakeID:       fake.id,
		fakeName:     fake.name,
		firstName:    first.name,
		participants: participants,
	}
	g.touch()
	g.notify()
	return nil
}

// Reveal shows the fake artist and the word to everyone.
func (g *Game) Reveal(byID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrGameNotFound
	}
	if byID != g.hostID {
		return ErrNotHost
	}
	if g.round == nil {
		return ErrNoRound
	}
	g.round.revealed = true
	g.touch()
	g.notify()
	return nil
}
