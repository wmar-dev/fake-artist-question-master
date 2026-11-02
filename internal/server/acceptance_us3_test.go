package server_test

import (
	"net/http"
	"testing"

	"fakeartist/internal/game"
)

func isRevealed(v game.View) bool { return v.Status == "revealed" }

func reveal(t *testing.T, host *player) {
	t.Helper()
	status, body := host.do("POST", "/api/games/"+host.code+"/reveal", "")
	if status != http.StatusNoContent {
		t.Fatalf("reveal: %d %s", status, body)
	}
}

// US3 scenario 1: revealing shows the same answer on every screen.
func TestRevealShowsTheAnswerOnEveryScreen(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	h.rng.Push(2, 0)
	dealRound(t, players[0])
	streams := make([]*stream, len(players))
	for i, p := range players {
		streams[i] = p.stream()
		streams[i].state()
	}
	word := players[0].stream().state().Round.Word

	reveal(t, players[0])

	for _, s := range streams {
		round := s.until(isRevealed).Round
		if round.FakeName != "Player2" || round.Word != word {
			t.Errorf("got %+v", round)
		}
	}
}

func TestRevealErrors(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	path := "/api/games/" + players[0].code + "/reveal"

	status, body := players[0].do("POST", path, "")
	wantError(t, status, body, 409, "no_round")

	dealRound(t, players[0])
	status, body = players[1].do("POST", path, "")
	wantError(t, status, body, 403, "not_host")
}

// US3 scenario 2: the same players get a new round.
func TestDealAgainAfterReveal(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	s := players[0].stream()
	s.state()

	dealRound(t, players[0])
	first := s.until(isDealt).Round
	reveal(t, players[0])
	s.until(isRevealed)
	dealRound(t, players[0])

	view := s.until(isDealt)
	if view.Round.FakeName != "" {
		t.Error("the new round shows the fake artist")
	}
	if view.Round.Number != first.Number+1 {
		t.Errorf("got round %d after round %d", view.Round.Number, first.Number)
	}
	if len(view.Players) != 4 {
		t.Errorf("got %d players", len(view.Players))
	}
}

// Spec edge case: the host can skip a word by dealing during a round.
func TestDealDuringARoundDiscardsIt(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	dealRound(t, players[0])
	first := players[0].stream().state().Round
	dealRound(t, players[0])
	second := players[0].stream().state().Round

	if second.Number != first.Number+1 {
		t.Errorf("got round %d after round %d", second.Number, first.Number)
	}
}

// US3 scenario 3: words do not repeat within a game.
func TestWordsDoNotRepeatAcrossRounds(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	seen := map[string]bool{}
	for range 100 {
		dealRound(t, players[0])
		reveal(t, players[0]) // Everyone can see the word, including the fake artist.
		word := players[0].stream().state().Round.Word
		if seen[word] {
			t.Fatalf("%q repeated", word)
		}
		seen[word] = true
	}
}
