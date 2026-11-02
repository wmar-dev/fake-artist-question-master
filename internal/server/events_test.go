package server_test

import (
	"testing"
	"time"

	"fakeartist/internal/game"
)

func TestStreamHeaders(t *testing.T) {
	s := newHarness(t).host("Sam").stream()
	s.state()
	for key, want := range map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"X-Accel-Buffering": "no",
	} {
		if got := s.header.Get(key); got != want {
			t.Errorf("%s: got %q, want %q", key, got, want)
		}
	}
}

func TestStreamSendsStateOnConnectAndChange(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	s := host.stream()

	if len(s.state().Players) != 1 {
		t.Error("first state is wrong")
	}
	h.join(host.code, "Alex")
	s.until(playerCount(2))
}

func TestStreamSendsKeepAliveEveryTwentyFiveSeconds(t *testing.T) {
	h := newHarness(t)
	s := h.host("Sam").stream()
	s.state()

	for range 3 {
		select {
		case e := <-s.events:
			t.Fatalf("got %+v before the keep-alive was due", e)
		case <-time.After(20 * time.Millisecond):
		}
		h.clock.Advance(25 * time.Second)
		if e := s.next(); e.name != "" {
			t.Fatalf("got %+v, want a keep-alive comment", e)
		}
	}
}

func TestIdleStreamSendsNothingElse(t *testing.T) {
	h := newHarness(t)
	s := h.host("Sam").stream()
	s.state()

	h.clock.Advance(24 * time.Second)
	select {
	case e := <-s.events:
		t.Fatalf("idle stream sent %+v", e)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestStreamReportsExpiry(t *testing.T) {
	h := newHarness(t)
	s := h.host("Sam").stream()
	s.state()

	h.clock.Advance(2 * time.Hour)
	if reason := s.closedReason(); reason != "expired" {
		t.Errorf("got reason %q", reason)
	}
}

func TestStreamRequiresAPlayer(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	status, body := h.browser().do("GET", "/api/games/"+host.code+"/events", "")
	wantError(t, status, body, 401, "not_a_player")
}

func TestCookieFromAnotherGameIsNotAccepted(t *testing.T) {
	h := newHarness(t)
	first, second := h.host("Sam"), h.host("Alex")
	status, body := first.do("GET", "/api/games/"+second.code+"/events", "")
	wantError(t, status, body, 401, "not_a_player")
}

func TestConnectionStatusIsShownToOthers(t *testing.T) {
	h := newHarness(t)
	host := h.host("Sam")
	alex := h.join(host.code, "Alex")
	s := host.stream()
	s.state()

	connected := func(want bool) func(game.View) bool {
		return func(v game.View) bool { return len(v.Players) == 2 && v.Players[1].Connected == want }
	}
	alexStream := alex.stream()
	s.until(connected(true))
	alexStream.state()
}
