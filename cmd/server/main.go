// Command server runs the Fake Artist Question Master website.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fakeartist/internal/game"
	"fakeartist/internal/server"
)

const shutdownWait = 5 * time.Second

type config struct {
	port      string
	publicURL string
	maxGames  int
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{port: "8080", maxGames: 200, publicURL: strings.TrimRight(getenv("PUBLIC_URL"), "/")}
	if v := getenv("PORT"); v != "" {
		if port, err := strconv.Atoi(v); err != nil || port < 1 || port > 65535 {
			return config{}, fmt.Errorf("PORT must be a port number, got %q", v)
		}
		cfg.port = v
	}
	if v := getenv("MAX_GAMES"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return config{}, fmt.Errorf("MAX_GAMES must be a positive number, got %q", v)
		}
		cfg.maxGames = n
	}
	if cfg.publicURL != "" {
		u, err := url.Parse(cfg.publicURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return config{}, fmt.Errorf("PUBLIC_URL must look like https://game.example.com, got %q", cfg.publicURL)
		}
	}
	return cfg, nil
}

func main() {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

func run(cfg config) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	clock := game.RealClock{}
	store := game.NewStore(clock, game.CryptoRand{}, game.Words(), cfg.maxGames)
	httpServer := &http.Server{
		Addr:              ":" + cfg.port,
		Handler:           server.New(server.Config{PublicURL: cfg.publicURL}, store, clock),
		ReadHeaderTimeout: 5 * time.Second,
	}

	failed := make(chan error, 1)
	go func() { failed <- httpServer.ListenAndServe() }()
	log.Printf("listening on port %s", cfg.port)

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	store.CloseAll(game.ReasonShutdown)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownWait)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}
