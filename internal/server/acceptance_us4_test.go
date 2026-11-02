package server_test

import (
	"testing"
	"time"

	"fakeartist/internal/game"
)

// US4 scenario 1: a change in one game does not reach another.
func TestGamesAreSeparate(t *testing.T) {
	h := newHarness(t)
	first, second := h.group(4), h.group(4)
	if first[0].code == second[0].code {
		t.Fatal("two games have the same code")
	}
	watcher := second[0].stream()
	before := watcher.state()

	dealRound(t, first[0])
	h.join(first[0].code, "Newcomer")

	select {
	case e := <-watcher.events:
		t.Errorf("the other game sent %+v", e)
	case <-time.After(100 * time.Millisecond):
	}
	if before.Status != "lobby" || len(before.Players) != 4 {
		t.Errorf("got %+v", before)
	}
}

// US4 scenario 2: being in one game gives no access to another.
func TestPlayersCannotActInAnotherGame(t *testing.T) {
	h := newHarness(t)
	first, second := h.group(4), h.group(4)
	path := "/api/games/" + second[0].code

	status, body := first[0].do("POST", path+"/rounds", "")
	wantError(t, status, body, 401, "not_a_player")

	status, body = first[0].do("POST", path+"/reveal", "")
	wantError(t, status, body, 401, "not_a_player")

	status, body = first[0].do("DELETE", path+"/players/"+second[1].id, "")
	wantError(t, status, body, 401, "not_a_player")
}

func TestEachGameDealsItsOwnRound(t *testing.T) {
	h := newHarness(t)
	first, second := h.group(4), h.group(4)
	firstStream, secondStream := first[0].stream(), second[0].stream()

	dealRound(t, first[0])
	dealRound(t, second[0])

	a := firstStream.until(isDealt).Round
	b := secondStream.until(isDealt).Round
	if a.Number != 1 || b.Number != 1 {
		t.Errorf("got rounds %d and %d", a.Number, b.Number)
	}
}

func TestCreatingPastTheLimitIsRefused(t *testing.T) {
	h := newHarness(t, options{maxGames: 2})
	h.host("Sam")
	h.host("Alex")

	status, body := h.browser().do("POST", "/api/games", `{"name":"Jo"}`)
	wantError(t, status, body, 503, "server_full")
}

func TestLimitFreesUpWhenGamesExpire(t *testing.T) {
	h := newHarness(t, options{maxGames: 1})
	h.host("Sam")
	h.clock.Advance(game.IdleLimit)
	h.host("Alex")
}
