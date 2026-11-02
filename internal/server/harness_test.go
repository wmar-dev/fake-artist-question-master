package server_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fakeartist/internal/game"
	"fakeartist/internal/gametest"
	"fakeartist/internal/server"
)

// updateWait is the longest a player may wait for a change (SC-003).
const updateWait = 2 * time.Second

type harness struct {
	t     *testing.T
	url   string
	clock *gametest.Clock
	rng   *gametest.Rand
}

type options struct {
	config   server.Config
	maxGames int
}

func newHarness(t *testing.T, opts ...options) *harness {
	t.Helper()
	opt := options{maxGames: 200}
	if len(opts) > 0 {
		opt = opts[0]
	}
	clock := gametest.NewClock()
	rng := gametest.NewRand(7)
	store := game.NewStore(clock, rng, game.Words(), opt.maxGames)
	ts := httptest.NewServer(server.New(opt.config, store, clock))
	t.Cleanup(ts.Close)
	return &harness{t: t, url: ts.URL, clock: clock, rng: rng}
}

// player is one browser, with its own cookies.
type player struct {
	h    *harness
	http *http.Client
	code string
	id   string
}

func (h *harness) browser() *player {
	jar, err := cookiejar.New(nil)
	if err != nil {
		h.t.Fatal(err)
	}
	return &player{h: h, http: &http.Client{Jar: jar}}
}

func (p *player) do(method, path, body string) (int, []byte) {
	p.h.t.Helper()
	req, err := http.NewRequest(method, p.h.url+path, strings.NewReader(body))
	if err != nil {
		p.h.t.Fatal(err)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		p.h.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		p.h.t.Fatal(err)
	}
	return resp.StatusCode, data
}

// host creates a game in a new browser.
func (h *harness) host(name string) *player {
	h.t.Helper()
	p := h.browser()
	status, body := p.do("POST", "/api/games", fmt.Sprintf(`{"name":%q}`, name))
	if status != http.StatusCreated {
		h.t.Fatalf("create game: %d %s", status, body)
	}
	var created struct{ Code string }
	if err := json.Unmarshal(body, &created); err != nil {
		h.t.Fatal(err)
	}
	p.code = created.Code
	return p
}

// join adds a player to the game in a new browser.
func (h *harness) join(code, name string) *player {
	h.t.Helper()
	p := h.browser()
	status, body := p.do("POST", "/api/games/"+code+"/players", fmt.Sprintf(`{"name":%q}`, name))
	if status != http.StatusCreated {
		h.t.Fatalf("join game: %d %s", status, body)
	}
	var joined struct{ PlayerID string }
	if err := json.Unmarshal(body, &joined); err != nil {
		h.t.Fatal(err)
	}
	p.code, p.id = code, joined.PlayerID
	return p
}

// group creates a game with the given number of players, host first.
func (h *harness) group(size int) []*player {
	h.t.Helper()
	players := []*player{h.host("Player0")}
	for i := 1; i < size; i++ {
		players = append(players, h.join(players[0].code, fmt.Sprintf("Player%d", i)))
	}
	return players
}

func wantError(t *testing.T, status int, body []byte, wantStatus int, wantCode string) {
	t.Helper()
	var got struct{ Error, Message string }
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("not a JSON error: %q", body)
	}
	if status != wantStatus || got.Error != wantCode {
		t.Errorf("got %d %q, want %d %q", status, got.Error, wantStatus, wantCode)
	}
	if got.Message == "" {
		t.Error("error has no message for the player")
	}
}

type event struct {
	name string // Empty for a keep-alive comment.
	data string
}

type stream struct {
	t      *testing.T
	header http.Header
	events chan event
}

// stream opens the player's event stream, closing it when the test ends.
func (p *player) stream() *stream {
	t := p.h.t
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, "GET", p.h.url+"/api/games/"+p.code+"/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("open stream: %d", resp.StatusCode)
	}
	s := &stream{t: t, header: resp.Header, events: make(chan event, 64)}
	go s.read(resp.Body)
	return s
}

func (s *stream) read(body io.ReadCloser) {
	defer close(s.events)
	defer body.Close()
	scanner := bufio.NewScanner(body)
	var e event
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, ":"):
			s.events <- event{data: strings.TrimSpace(line[1:])}
		case strings.HasPrefix(line, "event: "):
			e.name = line[len("event: "):]
		case strings.HasPrefix(line, "data: "):
			e.data = line[len("data: "):]
		case line == "":
			if e.name != "" {
				s.events <- e
			}
			e = event{}
		}
	}
}

func (s *stream) next() event {
	s.t.Helper()
	select {
	case e, ok := <-s.events:
		if !ok {
			s.t.Fatal("stream ended")
		}
		return e
	case <-time.After(updateWait):
		s.t.Fatal("no event within the time limit")
		return event{}
	}
}

// state returns the next state, ignoring keep-alives.
func (s *stream) state() game.View {
	s.t.Helper()
	for {
		e := s.next()
		if e.name == "" {
			continue
		}
		if e.name != "state" {
			s.t.Fatalf("got %q event, want state: %s", e.name, e.data)
		}
		var view game.View
		if err := json.Unmarshal([]byte(e.data), &view); err != nil {
			s.t.Fatal(err)
		}
		return view
	}
}

// until reads states until one matches, failing if none arrives in time.
func (s *stream) until(match func(game.View) bool) game.View {
	s.t.Helper()
	deadline := time.After(updateWait)
	for {
		select {
		case <-deadline:
			s.t.Fatal("no matching state within the time limit")
		default:
		}
		if view := s.state(); match(view) {
			return view
		}
	}
}

func (s *stream) closedReason() string {
	s.t.Helper()
	for {
		e := s.next()
		if e.name == "closed" {
			var closed struct{ Reason string }
			if err := json.Unmarshal([]byte(e.data), &closed); err != nil {
				s.t.Fatal(err)
			}
			return closed.Reason
		}
	}
}

func playerCount(n int) func(game.View) bool {
	return func(v game.View) bool { return len(v.Players) == n }
}
