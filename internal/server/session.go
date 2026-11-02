package server

import (
	"net/http"
	"strings"
	"time"

	"fakeartist/internal/game"
)

const (
	sessionCookie = "session"
	sessionMaxAge = 24 * time.Hour
)

// setSession stores a player's token in a cookie scoped to their game, so one
// browser can hold a place in more than one game.
func (s *Server) setSession(w http.ResponseWriter, code, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/api/games/" + code,
		MaxAge:   int(sessionMaxAge.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.PublicURL, "https://"),
	})
}

// authenticate finds the request's game and the player making the request.
func (s *Server) authenticate(r *http.Request) (*game.Game, string, error) {
	g, err := s.store.Get(r.PathValue("code"))
	if err != nil {
		return nil, "", err
	}
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil, "", errNotAPlayer
	}
	playerID, ok := g.PlayerByToken(cookie.Value)
	if !ok {
		return nil, "", errNotAPlayer
	}
	return g, playerID, nil
}

func (s *Server) isPlayer(r *http.Request, g *game.Game) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	_, ok := g.PlayerByToken(cookie.Value)
	return ok
}
