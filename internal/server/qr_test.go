package server

import (
	"bytes"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJoinAddress(t *testing.T) {
	tests := []struct {
		name          string
		publicURL     string
		host          string
		wantURL       string
		wantReachable bool
	}{
		{"network address", "", "192.168.1.5:8080", "http://192.168.1.5:8080/g/K7MQP", true},
		{"host name", "", "party.local:8080", "http://party.local:8080/g/K7MQP", true},
		{"localhost", "", "localhost:8080", "http://localhost:8080/g/K7MQP", false},
		{"localhost without port", "", "localhost", "http://localhost/g/K7MQP", false},
		{"loopback", "", "127.0.0.1:8080", "http://127.0.0.1:8080/g/K7MQP", false},
		{"other loopback", "", "127.1.2.3:8080", "http://127.1.2.3:8080/g/K7MQP", false},
		{"IPv6 loopback", "", "[::1]:8080", "http://[::1]:8080/g/K7MQP", false},
		{"all interfaces", "", "0.0.0.0:8080", "http://0.0.0.0:8080/g/K7MQP", false},
		{"public address wins", "https://game.example.com", "localhost:8080", "https://game.example.com/g/K7MQP", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestServer(Config{PublicURL: tt.publicURL})
			r := httptest.NewRequest("GET", "/", nil)
			r.Host = tt.host
			gotURL, gotReachable := s.joinAddress(r, "K7MQP")
			if gotURL != tt.wantURL || gotReachable != tt.wantReachable {
				t.Errorf("got %q, %v, want %q, %v", gotURL, gotReachable, tt.wantURL, tt.wantReachable)
			}
		})
	}
}

func TestJoinAddressIgnoresForwardedHeaders(t *testing.T) {
	s := newTestServer(Config{})
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "192.168.1.5:8080"
	r.Header.Set("X-Forwarded-Host", "evil.example.com")
	r.Header.Set("X-Forwarded-Proto", "https")
	got, _ := s.joinAddress(r, "K7MQP")
	if got != "http://192.168.1.5:8080/g/K7MQP" {
		t.Errorf("got %q", got)
	}
}

func TestQRImage(t *testing.T) {
	s := newTestServer(Config{})
	_, _, err := s.store.Create("Sam")
	if err != nil {
		t.Fatal(err)
	}
	g, _, _ := s.store.Create("Alex")

	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest("GET", "/api/games/"+g.Code()+"/qr.png", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("got content type %q", got)
	}
	if _, err := png.Decode(bytes.NewReader(rec.Body.Bytes())); err != nil {
		t.Errorf("not a PNG: %v", err)
	}
}

func TestQRImageForAnUnknownGame(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestServer(Config{}).ServeHTTP(rec, httptest.NewRequest("GET", "/api/games/ZZZZZ/qr.png", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("got %d", rec.Code)
	}
}
