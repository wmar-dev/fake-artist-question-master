package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// US1 scenario 1: creating a game makes the visitor the host.
func TestCreateGameMakesTheCreatorHost(t *testing.T) {
	h := newHarness(t)
	view := h.host("Sam").stream().state()

	if view.Status != "lobby" {
		t.Errorf("got status %q", view.Status)
	}
	if !view.You.IsHost || view.You.Name != "Sam" {
		t.Errorf("got you %+v", view.You)
	}
	if len(view.Players) != 1 || view.Players[0].Name != "Sam" || !view.Players[0].IsHost {
		t.Errorf("got players %+v", view.Players)
	}
	if view.MinPlayers != 4 || view.MaxPlayers != 10 {
		t.Errorf("got limits %d to %d", view.MinPlayers, view.MaxPlayers)
	}
}

// US1 scenarios 2 and 3: a QR code or typed code leads to the join page.
func TestJoinPageLooksUpTheGame(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	visitor := h.browser()

	for _, code := range []string{host.code, strings.ToLower(host.code)} {
		status, body := visitor.do("GET", "/api/games/"+code, "")
		var got struct {
			Code     string
			Joinable bool
			Joined   bool
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if status != http.StatusOK || got.Code != host.code || !got.Joinable || got.Joined {
			t.Errorf("code %s: got %d %+v", code, status, got)
		}
	}

	status, _ := visitor.do("GET", "/g/"+host.code, "")
	if status != http.StatusOK {
		t.Errorf("game page: got %d", status)
	}
}

func TestLookupShowsWhetherThisBrowserHasJoined(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")

	_, body := host.do("GET", "/api/games/"+host.code, "")
	var got struct{ Joined bool }
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Joined {
		t.Error("host's browser is not recognized as joined")
	}
}

func TestLookupShowsWhenAGameIsFull(t *testing.T) {
	h := newHarness(t)
	players := h.group(10)

	_, body := h.browser().do("GET", "/api/games/"+players[0].code, "")
	var got struct{ Joinable bool }
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.Joinable {
		t.Error("a full game says it is joinable")
	}
}

// US1 scenario 4: a player who joins appears on every other screen.
func TestJoiningUpdatesEveryScreen(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	hostStream := host.stream()
	hostStream.state()

	alex := h.join(host.code, "Alex")
	view := hostStream.until(playerCount(2))
	if view.Players[1].Name != "Alex" {
		t.Errorf("got players %+v", view.Players)
	}

	alexView := alex.stream().state()
	if alexView.You.IsHost || alexView.You.Name != "Alex" || len(alexView.Players) != 2 {
		t.Errorf("got %+v", alexView)
	}
}

func TestHostCanRemoveAPlayer(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	alex := h.join(host.code, "Alex")
	hostStream, alexStream := host.stream(), alex.stream()
	hostStream.state()
	alexStream.state()

	status, body := host.do("DELETE", "/api/games/"+host.code+"/players/"+alex.id, "")
	if status != http.StatusNoContent {
		t.Fatalf("got %d %s", status, body)
	}
	if reason := alexStream.closedReason(); reason != "removed" {
		t.Errorf("got reason %q", reason)
	}
	hostStream.until(playerCount(1))

	status, body = alex.do("GET", "/api/games/"+host.code+"/events", "")
	wantError(t, status, body, 401, "not_a_player")
}

func TestJoinErrors(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	join := func(code, name string) (int, []byte) {
		return h.browser().do("POST", "/api/games/"+code+"/players", fmt.Sprintf(`{"name":%q}`, name))
	}

	status, body := join(host.code, "")
	wantError(t, status, body, 400, "invalid_name")

	status, body = join(host.code, "sam")
	wantError(t, status, body, 409, "name_taken")

	status, body = join("ZZZZZ", "Alex")
	wantError(t, status, body, 404, "game_not_found")

	status, body = h.browser().do("POST", "/api/games/"+host.code+"/players", "not json")
	wantError(t, status, body, 400, "invalid_request")

	full := h.group(10)
	status, body = join(full[0].code, "Eleventh")
	wantError(t, status, body, 409, "game_full")
}

func TestCreateRejectsAnInvalidName(t *testing.T) {
	status, body := newHarness(t).browser().do("POST", "/api/games", `{"name":""}`)
	wantError(t, status, body, 400, "invalid_name")
}

func TestLookupOfAnUnknownGame(t *testing.T) {
	status, body := newHarness(t).browser().do("GET", "/api/games/ZZZZZ", "")
	wantError(t, status, body, 404, "game_not_found")
}

func TestRemoveErrors(t *testing.T) {
	h := newHarness(t)
	players := h.group(3)
	host, alex, sam := players[0], players[1], players[2]
	path := "/api/games/" + host.code + "/players/"

	status, body := alex.do("DELETE", path+sam.id, "")
	wantError(t, status, body, 403, "not_host")

	status, body = h.browser().do("DELETE", path+sam.id, "")
	wantError(t, status, body, 401, "not_a_player")

	status, body = host.do("DELETE", path+"nobody", "")
	wantError(t, status, body, 404, "player_not_found")

	view := host.stream().state()
	status, body = host.do("DELETE", path+view.You.ID, "")
	wantError(t, status, body, 409, "cannot_remove_host")
}
