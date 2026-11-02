package game

// View is everything one player may know about the game. It is the only game
// data sent to a browser, so secrets must be left out here, not hidden later.
type View struct {
	Code             string       `json:"code"`
	Status           string       `json:"status"`
	You              YouView      `json:"you"`
	Players          []PlayerView `json:"players"`
	MinPlayers       int          `json:"minPlayers"`
	MaxPlayers       int          `json:"maxPlayers"`
	CanClaimHost     bool         `json:"canClaimHost"`
	JoinURL          string       `json:"joinUrl"`
	JoinURLReachable bool         `json:"joinUrlReachable"`
	Round            *RoundView   `json:"round,omitempty"`
}

// YouView describes the player viewing the game.
type YouView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	IsHost bool   `json:"isHost"`
}

// PlayerView is what everyone can see about a player.
type PlayerView struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	IsHost    bool   `json:"isHost"`
}

// RoundView is what the player may know about the current round. Fields that
// must stay secret are left empty, so they are not sent at all.
type RoundView struct {
	Number      int    `json:"number,omitempty"`
	Category    string `json:"category,omitempty"`
	FirstPlayer string `json:"firstPlayer,omitempty"`
	Waiting     bool   `json:"waiting,omitempty"`
	IsFake      bool   `json:"isFake,omitempty"`
	Word        string `json:"word,omitempty"`
	FakeName    string `json:"fakeName,omitempty"`
}

// Game statuses.
const (
	StatusLobby    = "lobby"
	StatusDealt    = "dealt"
	StatusRevealed = "revealed"
)

// View builds the game as the given player may see it.
func (g *Game) View(playerID string) (View, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return View{}, ErrGameNotFound
	}
	you := g.findPlayer(playerID)
	if you == nil {
		return View{}, ErrPlayerNotFound
	}
	view := View{
		Code:         g.code,
		Status:       StatusLobby,
		You:          YouView{ID: you.id, Name: you.name, IsHost: you.id == g.hostID},
		MinPlayers:   MinPlayers,
		MaxPlayers:   MaxPlayers,
		CanClaimHost: you.id != g.hostID && g.hostGone(),
	}
	for _, p := range g.players {
		view.Players = append(view.Players, PlayerView{
			ID:        p.id,
			Name:      p.name,
			Connected: p.connections > 0,
			IsHost:    p.id == g.hostID,
		})
	}
	if g.round != nil {
		view.Status = StatusDealt
		if g.round.revealed {
			view.Status = StatusRevealed
		}
		view.Round = g.round.viewFor(you.id)
	}
	return view, nil
}

func (r *round) viewFor(playerID string) *RoundView {
	inRound := r.participants[playerID]
	if !inRound && !r.revealed {
		return &RoundView{Waiting: true}
	}
	view := &RoundView{Number: r.number, Category: r.category, FirstPlayer: r.firstName}
	isFake := playerID == r.fakeID
	view.IsFake = isFake
	if r.revealed {
		view.FakeName = r.fakeName
	}
	if r.revealed || !isFake {
		view.Word = r.word
	}
	return view
}
