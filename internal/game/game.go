// Package game holds the rules of the Question Master, with no HTTP or real clock.
package game

import (
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MinPlayers    = 4
	MaxPlayers    = 10
	maxNameLength = 20
	IdleLimit     = 2 * time.Hour
	HostWait      = time.Minute
	playerIDChars = 8
)

// Error is a failure a player can act on. Message is safe to show them.
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

var (
	ErrInvalidName      = &Error{"invalid_name", "Please enter a name of 1 to 20 characters."}
	ErrNameTaken        = &Error{"name_taken", "Someone in this game already has that name. Please choose another."}
	ErrGameFull         = &Error{"game_full", "This game already has 10 players. Ask the host to remove someone, or start a new game."}
	ErrGameNotFound     = &Error{"game_not_found", "We couldn't find that game. Check the code, or start a new game."}
	ErrPlayerNotFound   = &Error{"player_not_found", "That player has already left the game."}
	ErrNotHost          = &Error{"not_host", "Only the host can do that."}
	ErrCannotRemoveHost = &Error{"cannot_remove_host", "The host can't be removed from the game."}
	ErrServerFull       = &Error{"server_full", "The site is full right now. Please try again in a few minutes."}
	ErrWrongPlayerCount = &Error{"wrong_player_count", "You need at least 4 players to deal a round."}
	ErrNoRound          = &Error{"no_round", "There's no round to reveal yet. Deal one first."}
	ErrHostAvailable    = &Error{"host_available", "The host is still here, or has only just left. Give them a minute."}
)

// Reasons a Subscription ends.
const (
	ReasonRemoved  = "removed"
	ReasonExpired  = "expired"
	ReasonShutdown = "shutdown"
)

// Subscription delivers change signals for one open connection.
type Subscription struct {
	playerID string
	updates  chan struct{}
	done     chan struct{}
	reason   string
}

// Updates signals that the game changed.
func (s *Subscription) Updates() <-chan struct{} { return s.updates }

// Done is closed when the subscription ends.
func (s *Subscription) Done() <-chan struct{} { return s.done }

// Reason says why the subscription ended. Read it only after Done is closed.
func (s *Subscription) Reason() string { return s.reason }

// Identity is what a player holds after joining. Token is secret to them.
type Identity struct {
	ID    string
	Token string
}

type player struct {
	id             string
	token          string
	name           string
	connections    int
	disconnectedAt time.Time
}

// Game is one group's session. All of its methods are safe for concurrent use.
type Game struct {
	code    string
	clock   Clock
	rng     Rand
	words   []WordEntry
	onClose func(*Game)

	mu           sync.Mutex
	closed       bool
	hostID       string
	players      []*player
	lastActivity time.Time
	expiry       Timer
	handover     Timer
	subs         map[*Subscription]struct{}
	deck         []int
	nextWord     int
	round        *round
}

func newGame(code, hostName string, clock Clock, rng Rand, words []WordEntry, onClose func(*Game)) (*Game, Identity, error) {
	g := &Game{code: code, clock: clock, rng: rng, words: words, onClose: onClose, subs: map[*Subscription]struct{}{}}
	host, err := g.Join(hostName)
	if err != nil {
		return nil, Identity{}, err
	}
	g.hostID = host.ID
	g.deck = shuffledDeck(len(words), rng)
	g.watchHost()
	return g, host, nil
}

// Code returns the game's code.
func (g *Game) Code() string { return g.code }

// Join adds a player to the game.
func (g *Game) Join(name string) (Identity, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return Identity{}, ErrGameNotFound
	}
	name, err := cleanName(name)
	if err != nil {
		return Identity{}, err
	}
	if len(g.players) >= MaxPlayers {
		return Identity{}, ErrGameFull
	}
	for _, p := range g.players {
		if strings.EqualFold(p.name, name) {
			return Identity{}, ErrNameTaken
		}
	}
	p := &player{id: g.rng.Token()[:playerIDChars], token: g.rng.Token(), name: name, disconnectedAt: g.clock.Now()}
	g.players = append(g.players, p)
	g.touch()
	g.notify()
	return Identity{ID: p.id, Token: p.token}, nil
}

// Remove takes a player out of the game. Only the host can do this.
func (g *Game) Remove(byID, targetID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrGameNotFound
	}
	if byID != g.hostID {
		return ErrNotHost
	}
	if targetID == g.hostID {
		return ErrCannotRemoveHost
	}
	for i, p := range g.players {
		if p.id == targetID {
			g.players = append(g.players[:i], g.players[i+1:]...)
			g.closeSubs(ReasonRemoved, func(sub *Subscription) bool { return sub.playerID == targetID })
			g.touch()
			g.notify()
			return nil
		}
	}
	return ErrPlayerNotFound
}

// PlayerByToken returns the ID of the player holding token.
func (g *Game) PlayerByToken(token string) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range g.players {
		if p.token == token {
			return p.id, true
		}
	}
	return "", false
}

// Full reports whether the game has no room for another player.
func (g *Game) Full() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.players) >= MaxPlayers
}

// Subscribe connects a player to the game's updates.
func (g *Game) Subscribe(playerID string) (*Subscription, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return nil, ErrGameNotFound
	}
	p := g.findPlayer(playerID)
	if p == nil {
		return nil, ErrPlayerNotFound
	}
	sub := &Subscription{playerID: playerID, updates: make(chan struct{}, 1), done: make(chan struct{})}
	p.connections++
	g.watchHost()
	g.touch()
	g.notify() // Before adding sub, which sends its first state on its own.
	g.subs[sub] = struct{}{}
	return sub, nil
}

// Unsubscribe ends a player's connection to the game's updates.
func (g *Game) Unsubscribe(sub *Subscription) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.subs[sub]; !ok {
		return
	}
	delete(g.subs, sub)
	if p := g.findPlayer(sub.playerID); p != nil {
		p.connections--
		if p.connections == 0 {
			p.disconnectedAt = g.clock.Now()
		}
	}
	g.watchHost()
	g.notify()
}

// ClaimHost makes a player the host once the host has been away long enough.
func (g *Game) ClaimHost(playerID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return ErrGameNotFound
	}
	if playerID == g.hostID || !g.hostGone() {
		return ErrHostAvailable
	}
	g.hostID = playerID
	g.watchHost()
	g.touch()
	g.notify()
	return nil
}

func (g *Game) hostGone() bool {
	host := g.findPlayer(g.hostID)
	return host.connections == 0 && g.clock.Now().Sub(host.disconnectedAt) >= HostWait
}

// watchHost schedules a single update for when the host role becomes open, so
// waiting for it does no periodic work.
func (g *Game) watchHost() {
	if g.handover != nil {
		g.handover.Stop()
		g.handover = nil
	}
	host := g.findPlayer(g.hostID)
	if host.connections > 0 || g.closed {
		return
	}
	wait := HostWait - g.clock.Now().Sub(host.disconnectedAt)
	g.handover = g.clock.AfterFunc(max(wait, 0), func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.notify()
	})
}

// close ends the game for everyone connected.
func (g *Game) close(reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	g.closeSubs(reason, func(*Subscription) bool { return true })
	g.stopTimers()
}

func (g *Game) stopTimers() {
	if g.expiry != nil {
		g.expiry.Stop()
	}
	if g.handover != nil {
		g.handover.Stop()
	}
}

func (g *Game) findPlayer(id string) *player {
	for _, p := range g.players {
		if p.id == id {
			return p
		}
	}
	return nil
}

// notify tells each subscriber that the game changed. Updates coalesce, so a
// slow reader catches up with the latest state instead of holding up the game.
func (g *Game) notify() {
	for sub := range g.subs {
		select {
		case sub.updates <- struct{}{}:
		default:
		}
	}
}

func (g *Game) closeSubs(reason string, match func(*Subscription) bool) {
	for sub := range g.subs {
		if match(sub) {
			sub.reason = reason
			close(sub.done)
			delete(g.subs, sub)
		}
	}
}

func (g *Game) touch() {
	g.lastActivity = g.clock.Now()
	if g.expiry != nil {
		g.expiry.Stop()
	}
	g.expiry = g.clock.AfterFunc(IdleLimit, g.expire)
}

func (g *Game) expire() {
	g.mu.Lock()
	if g.closed || g.clock.Now().Sub(g.lastActivity) < IdleLimit {
		g.mu.Unlock()
		return
	}
	g.closed = true
	g.closeSubs(ReasonExpired, func(*Subscription) bool { return true })
	g.stopTimers()
	g.mu.Unlock()
	g.onClose(g)
}

func cleanName(name string) (string, error) {
	name = strings.TrimSpace(name)
	length := utf8.RuneCountInString(name)
	if length < 1 || length > maxNameLength || !utf8.ValidString(name) {
		return "", ErrInvalidName
	}
	if strings.ContainsFunc(name, unicode.IsControl) {
		return "", ErrInvalidName
	}
	return name, nil
}
