package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fakeartist/internal/game"
	"fakeartist/internal/gametest"
)

func newTestServer(cfg Config) *Server {
	clock := gametest.NewClock()
	store := game.NewStore(clock, gametest.NewRand(1), game.Words(), 200)
	return New(cfg, store, clock)
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(Config{}).ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestPagesAreServed(t *testing.T) {
	s := newTestServer(Config{})
	for _, path := range []string{"/", "/g/K7MQP", "/static/app.js", "/static/style.css", "/static/favicon.svg"} {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: got %d", path, rec.Code)
		}
	}
}

func TestWriteErrorUsesStatusAndBody(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{game.ErrInvalidName, 400, "invalid_name"},
		{game.ErrNotHost, 403, "not_host"},
		{game.ErrGameNotFound, 404, "game_not_found"},
		{game.ErrGameFull, 409, "game_full"},
		{game.ErrServerFull, 503, "server_full"},
		{io.ErrUnexpectedEOF, 500, "internal"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		writeError(rec, tt.err)
		if rec.Code != tt.status {
			t.Errorf("%v: got status %d, want %d", tt.err, rec.Code, tt.status)
		}
		var body struct{ Error, Message string }
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%v: %v", tt.err, err)
		}
		if body.Error != tt.code || body.Message == "" {
			t.Errorf("%v: got body %+v", tt.err, body)
		}
	}
}

func TestReadJSONLimitsBodySize(t *testing.T) {
	var dst struct{ Name string }
	small := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Sam"}`))
	if err := readJSON(httptest.NewRecorder(), small, &dst); err != nil {
		t.Fatalf("valid body refused: %v", err)
	}

	big := `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`
	large := httptest.NewRequest("POST", "/", strings.NewReader(big))
	if err := readJSON(httptest.NewRecorder(), large, &dst); err != errInvalidRequest {
		t.Errorf("got %v, want errInvalidRequest", err)
	}

	bad := httptest.NewRequest("POST", "/", strings.NewReader("not json"))
	if err := readJSON(httptest.NewRecorder(), bad, &dst); err != errInvalidRequest {
		t.Errorf("got %v, want errInvalidRequest", err)
	}
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	s := newTestServer(Config{})
	for _, path := range []string{"/", "/g/K7MQP", "/static/app.js", "/healthz", "/api/games/ZZZZZ"} {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		policy := rec.Header().Get("Content-Security-Policy")
		for _, want := range []string{"default-src 'none'", "script-src 'self'", "img-src 'self'", "frame-ancestors 'none'"} {
			if !strings.Contains(policy, want) {
				t.Errorf("%s: policy %q is missing %q", path, policy, want)
			}
		}
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s: got X-Content-Type-Options %q", path, got)
		}
		if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("%s: got Referrer-Policy %q", path, got)
		}
	}
}

func TestPagesAllowTheirOwnFilesOnly(t *testing.T) {
	s := newTestServer(Config{})
	for _, path := range []string{"/", "/g/K7MQP"} {
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		body := rec.Body.String()
		if strings.Contains(body, "<style") || strings.Contains(body, "style=") || strings.Contains(body, "<script>") {
			t.Errorf("%s has inline code that the security policy would block", path)
		}
	}
}
