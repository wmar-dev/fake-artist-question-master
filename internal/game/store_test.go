package game_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"fakeartist/internal/game"
	"fakeartist/internal/gametest"
)

func TestCreateMakesAFiveCharacterCode(t *testing.T) {
	store, _, _ := newStore(t)
	for range 50 {
		g, _, err := store.Create("Host")
		if err != nil {
			t.Fatal(err)
		}
		code := g.Code()
		if len(code) != 5 {
			t.Fatalf("code %q is not 5 characters", code)
		}
		if strings.ContainsAny(code, "01OIL") {
			t.Fatalf("code %q has a look-alike character", code)
		}
	}
}

func TestGetIgnoresCase(t *testing.T) {
	store, _, _ := newStore(t)
	g, _, _ := store.Create("Host")
	got, err := store.Get(strings.ToLower(g.Code()))
	if err != nil {
		t.Fatal(err)
	}
	if got != g {
		t.Error("got a different game")
	}
}

func TestGetUnknownCode(t *testing.T) {
	store, _, _ := newStore(t)
	_, err := store.Get("ZZZZZ")
	wantErr(t, err, game.ErrGameNotFound)
}

func TestCreateRefusedAtCapacity(t *testing.T) {
	store := game.NewStore(gametest.NewClock(), gametest.NewRand(1), testWords, 2)
	for range 2 {
		if _, _, err := store.Create("Host"); err != nil {
			t.Fatal(err)
		}
	}
	_, _, err := store.Create("Host")
	wantErr(t, err, game.ErrServerFull)
}

func TestCreateRejectsAnInvalidHostName(t *testing.T) {
	store, _, _ := newStore(t)
	_, _, err := store.Create("")
	wantErr(t, err, game.ErrInvalidName)
	if store.Count() != 0 {
		t.Error("a game was kept after the name was rejected")
	}
}

func TestGameExpiresAfterTwoHoursWithoutActivity(t *testing.T) {
	store, clock, _ := newStore(t)
	g, _, _ := store.Create("Host")

	clock.Advance(2*time.Hour - time.Second)
	if _, err := store.Get(g.Code()); err != nil {
		t.Fatalf("game expired early: %v", err)
	}

	clock.Advance(time.Second)
	_, err := store.Get(g.Code())
	wantErr(t, err, game.ErrGameNotFound)
	if store.Count() != 0 {
		t.Errorf("got %d games, want 0", store.Count())
	}
}

func TestActivityDelaysExpiry(t *testing.T) {
	store, clock, _ := newStore(t)
	g, _, _ := store.Create("Host")

	clock.Advance(90 * time.Minute)
	if _, err := g.Join("Alex"); err != nil {
		t.Fatal(err)
	}
	clock.Advance(90 * time.Minute)
	if _, err := store.Get(g.Code()); err != nil {
		t.Fatalf("game expired despite activity: %v", err)
	}
}

func TestExpiredGameFreesCapacity(t *testing.T) {
	clock := gametest.NewClock()
	store := game.NewStore(clock, gametest.NewRand(1), testWords, 1)
	g, _, _ := store.Create("Host")

	clock.Advance(2 * time.Hour)
	if _, _, err := store.Create("Host"); err != nil {
		t.Fatalf("capacity was not freed: %v", err)
	}
	_, err := g.Join("Alex")
	wantErr(t, err, game.ErrGameNotFound)
}

// TestConcurrentUse is most useful under go test -race.
func TestConcurrentUse(t *testing.T) {
	store, clock, _ := newStore(t)
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g, host, err := store.Create(fmt.Sprintf("Host%d", i))
			if err != nil {
				t.Error(err)
				return
			}
			var joined sync.WaitGroup
			for j := range 5 {
				joined.Add(1)
				go func() {
					defer joined.Done()
					p, err := g.Join(fmt.Sprintf("Player%d", j))
					if err != nil {
						t.Error(err)
						return
					}
					sub, err := g.Subscribe(p.ID)
					if err != nil {
						t.Error(err)
						return
					}
					defer g.Unsubscribe(sub)
					if _, err := g.View(p.ID); err != nil {
						t.Error(err)
					}
				}()
			}
			joined.Wait()
			for range 5 {
				if err := g.Deal(host.ID); err != nil {
					t.Error(err)
				}
				if err := g.Reveal(host.ID); err != nil {
					t.Error(err)
				}
				if _, err := store.Get(g.Code()); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()

	clock.Advance(2 * time.Hour)
	if store.Count() != 0 {
		t.Errorf("got %d games after expiry", store.Count())
	}
}
