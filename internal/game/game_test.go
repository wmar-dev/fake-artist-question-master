package game_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"fakeartist/internal/game"
	"fakeartist/internal/gametest"
)

var testWords = []game.WordEntry{
	{Category: "Food", Word: "Tomato"},
	{Category: "Food", Word: "Pizza"},
	{Category: "Animal", Word: "Horse"},
	{Category: "Animal", Word: "Whale"},
}

func newStore(t *testing.T) (*game.Store, *gametest.Clock, *gametest.Rand) {
	t.Helper()
	clock := gametest.NewClock()
	rng := gametest.NewRand(1)
	return game.NewStore(clock, rng, testWords, 200), clock, rng
}

func newGame(t *testing.T) (*game.Game, game.Identity) {
	t.Helper()
	store, _, _ := newStore(t)
	g, host, err := store.Create("Host")
	if err != nil {
		t.Fatal(err)
	}
	return g, host
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func TestJoinNameRules(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
	}{
		{"valid", "Alex", nil},
		{"trimmed", "  Alex  ", nil},
		{"twenty characters", strings.Repeat("a", 20), nil},
		{"twenty multibyte characters", strings.Repeat("é", 20), nil},
		{"empty", "", game.ErrInvalidName},
		{"only spaces", "   ", game.ErrInvalidName},
		{"too long", strings.Repeat("a", 21), game.ErrInvalidName},
		{"control character", "Al\nex", game.ErrInvalidName},
		{"invalid encoding", "Al\xffex", game.ErrInvalidName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, _ := newGame(t)
			_, err := g.Join(tt.input)
			wantErr(t, err, tt.want)
		})
	}
}

func TestJoinNameIsUniqueIgnoringCase(t *testing.T) {
	g, _ := newGame(t)
	if _, err := g.Join("Alex"); err != nil {
		t.Fatal(err)
	}
	_, err := g.Join(" aLEX ")
	wantErr(t, err, game.ErrNameTaken)
}

func TestJoinRefusesAnEleventhPlayer(t *testing.T) {
	g, _ := newGame(t)
	for i := range 9 {
		if _, err := g.Join(string(rune('a' + i))); err != nil {
			t.Fatal(err)
		}
	}
	_, err := g.Join("eleventh")
	wantErr(t, err, game.ErrGameFull)
}

func TestHostCanRemoveAPlayer(t *testing.T) {
	g, host := newGame(t)
	alex, err := g.Join("Alex")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Remove(host.ID, alex.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.PlayerByToken(alex.Token); ok {
		t.Error("removed player's token still works")
	}
	if _, err := g.Join("Alex"); err != nil {
		t.Errorf("name should be free after removal: %v", err)
	}
}

func TestRemoveRules(t *testing.T) {
	g, host := newGame(t)
	alex, _ := g.Join("Alex")
	sam, _ := g.Join("Sam")

	wantErr(t, g.Remove(alex.ID, sam.ID), game.ErrNotHost)
	wantErr(t, g.Remove(host.ID, host.ID), game.ErrCannotRemoveHost)
	wantErr(t, g.Remove(host.ID, "nobody"), game.ErrPlayerNotFound)
}

func TestPlayerByToken(t *testing.T) {
	g, host := newGame(t)
	id, ok := g.PlayerByToken(host.Token)
	if !ok || id != host.ID {
		t.Errorf("got %q, %v, want %q, true", id, ok, host.ID)
	}
	if _, ok := g.PlayerByToken("wrong"); ok {
		t.Error("unknown token was accepted")
	}
}

func handoverGame(t *testing.T) (*game.Game, game.Identity, game.Identity, *gametest.Clock) {
	t.Helper()
	store, clock, _ := newStore(t)
	g, host, err := store.Create("Host")
	if err != nil {
		t.Fatal(err)
	}
	alex, err := g.Join("Alex")
	if err != nil {
		t.Fatal(err)
	}
	return g, host, alex, clock
}

func subscribe(t *testing.T, g *game.Game, p game.Identity) *game.Subscription {
	t.Helper()
	sub, err := g.Subscribe(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return sub
}

func TestHostHandoverOpensAfterOneMinuteAway(t *testing.T) {
	g, host, alex, clock := handoverGame(t)
	hostSub := subscribe(t, g, host)
	alexSub := subscribe(t, g, alex)
	g.Unsubscribe(hostSub)

	clock.Advance(game.HostWait - time.Second)
	if viewOf(t, g, alex).CanClaimHost {
		t.Fatal("handover opened early")
	}
	wantErr(t, g.ClaimHost(alex.ID), game.ErrHostAvailable)

	select {
	case <-alexSub.Updates():
	default:
	}
	clock.Advance(time.Second)
	select {
	case <-alexSub.Updates():
	default:
		t.Error("players were not told the host role is open")
	}
	if !viewOf(t, g, alex).CanClaimHost {
		t.Error("handover did not open")
	}
	if viewOf(t, g, host).CanClaimHost {
		t.Error("the host is offered their own role")
	}
}

func TestClaimingHostMovesTheRole(t *testing.T) {
	g, host, alex, clock := handoverGame(t)
	clock.Advance(game.HostWait)

	if err := g.ClaimHost(alex.ID); err != nil {
		t.Fatal(err)
	}
	if !viewOf(t, g, alex).You.IsHost {
		t.Error("claimant is not the host")
	}
	if viewOf(t, g, host).You.IsHost {
		t.Error("old host is still the host")
	}
	if viewOf(t, g, alex).CanClaimHost {
		t.Error("the new host is offered the role")
	}
	wantErr(t, g.Remove(host.ID, alex.ID), game.ErrNotHost)
}

func TestReturningHostCancelsHandover(t *testing.T) {
	g, host, alex, clock := handoverGame(t)
	g.Unsubscribe(subscribe(t, g, host))
	clock.Advance(game.HostWait - time.Second)

	subscribe(t, g, host)
	clock.Advance(time.Hour)
	if viewOf(t, g, alex).CanClaimHost {
		t.Error("handover is open while the host is connected")
	}
	wantErr(t, g.ClaimHost(alex.ID), game.ErrHostAvailable)
}

func TestHostWithTwoConnectionsStaysUntilBothClose(t *testing.T) {
	g, host, alex, clock := handoverGame(t)
	first, second := subscribe(t, g, host), subscribe(t, g, host)
	g.Unsubscribe(first)

	clock.Advance(game.HostWait)
	if viewOf(t, g, alex).CanClaimHost {
		t.Error("handover opened while the host still has a connection")
	}
	g.Unsubscribe(second)
	clock.Advance(game.HostWait)
	if !viewOf(t, g, alex).CanClaimHost {
		t.Error("handover did not open")
	}
}

func TestCloseEndsSubscriptions(t *testing.T) {
	store, _, _ := newStore(t)
	g, host, _ := store.Create("Host")
	sub := subscribe(t, g, host)

	store.CloseAll(game.ReasonShutdown)

	select {
	case <-sub.Done():
	default:
		t.Fatal("subscription is still open")
	}
	if sub.Reason() != game.ReasonShutdown {
		t.Errorf("got reason %q", sub.Reason())
	}
	if store.Count() != 0 {
		t.Errorf("got %d games", store.Count())
	}
}
