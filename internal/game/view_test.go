package game_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"fakeartist/internal/game"
)

func marshalled(t *testing.T, view game.View) string {
	t.Helper()
	data, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// dealtTable deals a round with a known fake artist at index 1.
func dealtTable(t *testing.T) (*game.Game, []game.Identity) {
	t.Helper()
	g, players, rng := table(t, 5, testWords)
	rng.Push(1, 2)
	deal(t, g, players[0])
	return g, players
}

func TestLobbyViewHasNoRound(t *testing.T) {
	g, players, _ := table(t, 4, testWords)
	view := viewOf(t, g, players[0])
	if view.Status != game.StatusLobby || view.Round != nil {
		t.Errorf("got %+v", view)
	}
	if strings.Contains(marshalled(t, view), `"round"`) {
		t.Error("lobby view sends a round")
	}
}

func TestArtistViewBeforeReveal(t *testing.T) {
	g, players := dealtTable(t)
	round := viewOf(t, g, players[0]).Round

	if round.Word == "" || round.Category == "" || round.FirstPlayer == "" {
		t.Errorf("got %+v", round)
	}
	if round.IsFake || round.Waiting {
		t.Errorf("got %+v", round)
	}
	if round.FakeName != "" {
		t.Error("artist can see the fake artist")
	}
}

func TestFakeViewBeforeReveal(t *testing.T) {
	g, players := dealtTable(t)
	artistRound := viewOf(t, g, players[0]).Round
	view := viewOf(t, g, players[1])

	if !view.Round.IsFake || view.Round.Category == "" || view.Round.FirstPlayer == "" {
		t.Errorf("got %+v", view.Round)
	}
	if view.Round.Word != "" {
		t.Error("fake artist's view has the word")
	}
	if strings.Contains(marshalled(t, view), artistRound.Word) {
		t.Errorf("the word %q is somewhere in the fake artist's data", artistRound.Word)
	}
}

func TestLateJoinerViewBeforeReveal(t *testing.T) {
	g, players := dealtTable(t)
	artistRound := viewOf(t, g, players[0]).Round
	late, err := g.Join("Late")
	if err != nil {
		t.Fatal(err)
	}
	view := viewOf(t, g, late)

	if !reflect.DeepEqual(*view.Round, game.RoundView{Waiting: true}) {
		t.Errorf("got %+v, want only waiting", view.Round)
	}
	data := marshalled(t, view)
	if strings.Contains(data, artistRound.Word) || strings.Contains(data, artistRound.Category) {
		t.Error("late joiner's data has round details")
	}
}

func TestArtistsCannotTellAnythingFromTheirData(t *testing.T) {
	g, players := dealtTable(t)
	first, second := viewOf(t, g, players[0]), viewOf(t, g, players[2])
	first.You, second.You = game.YouView{}, game.YouView{}
	if !reflect.DeepEqual(first, second) {
		t.Errorf("artists' data differs:\n%+v\n%+v", first, second)
	}
}

func TestOutsiderHasNoView(t *testing.T) {
	g, _ := dealtTable(t)
	_, err := g.View("nobody")
	wantErr(t, err, game.ErrPlayerNotFound)
}

func TestLateJoinerSeesTheAnswerAfterReveal(t *testing.T) {
	g, players := dealtTable(t)
	word := viewOf(t, g, players[0]).Round.Word
	late, err := g.Join("Late")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Reveal(players[0].ID); err != nil {
		t.Fatal(err)
	}

	round := viewOf(t, g, late).Round
	if round.Waiting || round.Word != word || round.FakeName != "Player1" {
		t.Errorf("got %+v", round)
	}
}

func TestFakeArtistSeesTheWordAfterReveal(t *testing.T) {
	g, players := dealtTable(t)
	word := viewOf(t, g, players[0]).Round.Word
	if err := g.Reveal(players[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := viewOf(t, g, players[1]).Round.Word; got != word {
		t.Errorf("got %q, want %q", got, word)
	}
}
