package server_test

import (
	"context"
	"net/http"
	"testing"

	"fakeartist/internal/game"
)

func canClaim(v game.View) bool { return v.CanClaimHost }

// Spec edge case: the game carries on if the host stays away.
func TestAnotherPlayerCanTakeOverAsHost(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	host, alex := players[0], players[1]
	s := alex.stream()
	s.state()

	path := "/api/games/" + host.code + "/host"
	status, body := alex.do("POST", path, "")
	wantError(t, status, body, 409, "host_available")

	h.clock.Advance(game.HostWait) // The host never opened a connection.
	s.until(canClaim)

	status, body = h.browser().do("POST", path, "")
	wantError(t, status, body, 401, "not_a_player")

	status, body = alex.do("POST", path, "")
	if status != http.StatusNoContent {
		t.Fatalf("claim: %d %s", status, body)
	}
	view := s.until(func(v game.View) bool { return v.You.IsHost })
	if view.CanClaimHost {
		t.Error("the new host is still offered the role")
	}
	dealRound(t, alex)

	status, body = host.do("POST", "/api/games/"+host.code+"/rounds", "")
	wantError(t, status, body, 403, "not_host")
}

func TestDroppedConnectionStartsTheHandoverWait(t *testing.T) {
	h := newHarness(t)
	players := h.group(4)
	s := players[1].stream()
	s.state()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", h.url+"/api/games/"+players[0].code+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := players[0].http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	hostConnected := func(want bool) func(game.View) bool {
		return func(v game.View) bool { return v.Players[0].Connected == want }
	}
	s.until(hostConnected(true))

	cancel()
	s.until(hostConnected(false))
	h.clock.Advance(game.HostWait)
	s.until(canClaim)
}
