// Package server exposes the game over HTTP.
package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"fakeartist/internal/game"
	"fakeartist/web"
)

const (
	maxBodyBytes = 1024

	// Pages use only their own files, so nothing else may load or run.
	contentSecurityPolicy = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; " +
		"connect-src 'self'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
)

var (
	errInvalidRequest = &game.Error{Code: "invalid_request", Message: "That request wasn't valid. Please reload the page and try again."}
	errNotAPlayer     = &game.Error{Code: "not_a_player", Message: "You're not in this game. Use the game's link or code to join."}
	errInternal       = &game.Error{Code: "internal", Message: "Something went wrong. Please try again."}
)

var errorStatus = map[string]int{
	"invalid_name":       http.StatusBadRequest,
	"invalid_request":    http.StatusBadRequest,
	"not_a_player":       http.StatusUnauthorized,
	"not_host":           http.StatusForbidden,
	"game_not_found":     http.StatusNotFound,
	"player_not_found":   http.StatusNotFound,
	"name_taken":         http.StatusConflict,
	"game_full":          http.StatusConflict,
	"wrong_player_count": http.StatusConflict,
	"no_round":           http.StatusConflict,
	"host_available":     http.StatusConflict,
	"cannot_remove_host": http.StatusConflict,
	"server_full":        http.StatusServiceUnavailable,
}

// Config holds deployment settings.
type Config struct {
	// PublicURL is the base address for join links. If empty, the address of
	// each request is used.
	PublicURL string
}

// Server handles all HTTP requests.
type Server struct {
	cfg     Config
	store   *game.Store
	clock   game.Clock
	handler http.Handler
}

// New returns a Server with its routes registered.
func New(cfg Config, store *game.Store, clock game.Clock) *Server {
	s := &Server{cfg: cfg, store: store, clock: clock}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /{$}", page("index.html"))
	mux.HandleFunc("GET /g/{code}", page("game.html"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(web.FS)))
	mux.HandleFunc("POST /api/games", s.createGame)
	mux.HandleFunc("GET /api/games/{code}", s.lookupGame)
	mux.HandleFunc("POST /api/games/{code}/players", s.joinGame)
	mux.HandleFunc("DELETE /api/games/{code}/players/{id}", s.removePlayer)
	mux.HandleFunc("POST /api/games/{code}/rounds", s.dealRound)
	mux.HandleFunc("POST /api/games/{code}/reveal", s.revealRound)
	mux.HandleFunc("POST /api/games/{code}/host", s.claimHost)
	mux.HandleFunc("GET /api/games/{code}/qr.png", s.qrImage)
	mux.HandleFunc("GET /api/games/{code}/events", s.events)
	s.handler = secureHeaders(mux)
	return s
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Content-Security-Policy", contentSecurityPolicy)
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

type nameRequest struct {
	Name string `json:"name"`
}

func (s *Server) createGame(w http.ResponseWriter, r *http.Request) {
	var req nameRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	g, host, err := s.store.Create(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	s.setSession(w, g.Code(), host.Token)
	writeJSON(w, http.StatusCreated, map[string]string{"code": g.Code()})
}

func (s *Server) lookupGame(w http.ResponseWriter, r *http.Request) {
	g, err := s.store.Get(r.PathValue("code"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code":     g.Code(),
		"joinable": !g.Full(),
		"joined":   s.isPlayer(r, g),
	})
}

func (s *Server) joinGame(w http.ResponseWriter, r *http.Request) {
	g, err := s.store.Get(r.PathValue("code"))
	if err != nil {
		writeError(w, err)
		return
	}
	var req nameRequest
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	p, err := g.Join(req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	s.setSession(w, g.Code(), p.Token)
	writeJSON(w, http.StatusCreated, map[string]string{"playerId": p.ID})
}

func (s *Server) removePlayer(w http.ResponseWriter, r *http.Request) {
	g, playerID, err := s.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := g.Remove(playerID, r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) dealRound(w http.ResponseWriter, r *http.Request) {
	g, playerID, err := s.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := g.Deal(playerID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) revealRound(w http.ResponseWriter, r *http.Request) {
	g, playerID, err := s.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := g.Reveal(playerID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) claimHost(w http.ResponseWriter, r *http.Request) {
	g, playerID, err := s.authenticate(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := g.ClaimHost(playerID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, web.FS, name)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	var gameErr *game.Error
	if !errors.As(err, &gameErr) {
		gameErr = errInternal
	}
	status, ok := errorStatus[gameErr.Code]
	if !ok {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, map[string]string{"error": gameErr.Code, "message": gameErr.Message})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		return errInvalidRequest
	}
	return nil
}
