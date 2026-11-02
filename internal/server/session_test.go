package server_test

import (
	"net/http"
	"strings"
	"testing"

	"fakeartist/internal/server"
)

func sessionCookie(t *testing.T, publicURL string) *http.Cookie {
	t.Helper()
	h := newHarness(t, options{config: server.Config{PublicURL: publicURL}, maxGames: 200})
	resp, err := http.Post(h.url+"/api/games", "application/json", strings.NewReader(`{"name":"Sam"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	return cookies[0]
}

func TestSessionCookieIsLockedDown(t *testing.T) {
	cookie := sessionCookie(t, "")
	if !cookie.HttpOnly {
		t.Error("cookie is readable by scripts")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("got SameSite %v", cookie.SameSite)
	}
	if !strings.HasPrefix(cookie.Path, "/api/games/") || len(cookie.Path) <= len("/api/games/") {
		t.Errorf("cookie is not scoped to the game: path %q", cookie.Path)
	}
	if cookie.Secure {
		t.Error("cookie is Secure on plain HTTP, so local hosting would break")
	}
}

func TestSessionCookieIsSecureOverHTTPS(t *testing.T) {
	if !sessionCookie(t, "https://game.example.com").Secure {
		t.Error("cookie is not Secure with an HTTPS public address")
	}
}

func TestSessionCookieIsNotSecureOverPlainHTTPAddress(t *testing.T) {
	if sessionCookie(t, "http://192.168.1.5:8080").Secure {
		t.Error("cookie is Secure with an HTTP public address")
	}
}
