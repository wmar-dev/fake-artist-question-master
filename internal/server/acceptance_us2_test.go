package server_test

import (
	"net/http"
	"strings"
	"testing"

	"fakeartist/internal/game"
)

func isDealt(v game.View) bool { return v.Status == "dealt" }

func dealRound(t *testing.T, host *player) {
	t.Helper()
	status, body := host.do("POST", "/api/games/"+host.code+"/rounds", "")
	if status != http.StatusCreated {
		t.Fatalf("deal: %d %s", status, body)
	}
}

// US2 scenarios 1 to 3: everyone gets the right role from a deal.
func TestDealGivesEveryPlayerTheirRole(t *testing.T) {
	h := newHarness(t)
	players := h.group(5)
	streams := make([]*stream, len(players))
	for i, p := range players {
		streams[i] = p.stream()
		streams[i].state()
	}

	dealRound(t, players[0])

	fakes := 0
	words := map[string]bool{}
	categories := map[string]bool{}
	firstPlayers := map[string]bool{}
	for _, s := range streams {
		round := s.until(isDealt).Round
		categories[round.Category] = true
		firstPlayers[round.FirstPlayer] = true
		if round.IsFake {
			fakes++
			if round.Word != "" {
				t.Error("fake artist received the word")
			}
		} else {
			words[round.Word] = true
		}
		if round.FakeName != "" {
			t.Error("the fake artist's name was sent before the reveal")
		}
	}
	if fakes != 1 {
		t.Errorf("got %d fake artists", fakes)
	}
	if len(words) != 1 || words[""] {
		t.Errorf("got words %v", words)
	}
	if len(categories) != 1 || len(firstPlayers) != 1 {
		t.Errorf("got categories %v and first players %v", categories, firstPlayers)
	}
}

func TestFakeArtistStreamNeverContainsTheWord(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	streams := make([]*stream, len(players))
	for i, p := range players {
		streams[i] = p.stream()
		streams[i].state()
	}
	h.rng.Push(2, 0)
	dealRound(t, players[0])

	word := streams[0].until(isDealt).Round.Word
	for {
		raw := streams[2].next()
		if strings.Contains(raw.data, word) {
			t.Fatalf("raw event for the fake artist contains %q", word)
		}
		if strings.Contains(raw.data, `"status":"dealt"`) {
			if !strings.Contains(raw.data, `"isFake":true`) {
				t.Errorf("expected the fake artist's event, got %s", raw.data)
			}
			return
		}
	}
}

func TestDealNeedsFourPlayers(t *testing.T) {
	h := newHarness(t)
	players := h.group(3)
	status, body := players[0].do("POST", "/api/games/"+players[0].code+"/rounds", "")
	wantError(t, status, body, 409, "wrong_player_count")
}

func TestOnlyTheHostCanDeal(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	status, body := players[1].do("POST", "/api/games/"+players[1].code+"/rounds", "")
	wantError(t, status, body, 403, "not_host")

	status, body = h.browser().do("POST", "/api/games/"+players[1].code+"/rounds", "")
	wantError(t, status, body, 401, "not_a_player")
}

// FR-020: a player who reconnects gets the same role back.
func TestReconnectingRestoresTheSameRole(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	dealRound(t, players[0])

	for _, p := range players {
		before := p.stream().state().Round
		after := p.stream().state().Round
		if *before != *after {
			t.Errorf("role changed from %+v to %+v", before, after)
		}
	}
}

// Spec edge case: someone who joins during a round waits for the next one.
func TestLateJoinerWaitsForTheNextRound(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	dealRound(t, players[0])

	late := h.join(players[0].code, "Late")
	s := late.stream()
	if round := s.state().Round; !round.Waiting || round.Word != "" || round.Category != "" {
		t.Errorf("got %+v", round)
	}

	dealRound(t, players[0])
	round := s.until(func(v game.View) bool { return !v.Round.Waiting }).Round
	if round.Category == "" {
		t.Errorf("late joiner has no role in the next round: %+v", round)
	}
}
