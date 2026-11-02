package game_test

import (
	"fmt"
	"testing"

	"fakeartist/internal/game"
	"fakeartist/internal/gametest"
)

// table creates a game with size players, host first.
func table(t *testing.T, size int, words []game.WordEntry) (*game.Game, []game.Identity, *gametest.Rand) {
	t.Helper()
	rng := gametest.NewRand(3)
	store := game.NewStore(gametest.NewClock(), rng, words, 200)
	g, host, err := store.Create("Player0")
	if err != nil {
		t.Fatal(err)
	}
	players := []game.Identity{host}
	for i := 1; i < size; i++ {
		p, err := g.Join(fmt.Sprintf("Player%d", i))
		if err != nil {
			t.Fatal(err)
		}
		players = append(players, p)
	}
	return g, players, rng
}

func viewOf(t *testing.T, g *game.Game, p game.Identity) game.View {
	t.Helper()
	view, err := g.View(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	return view
}

func deal(t *testing.T, g *game.Game, host game.Identity) {
	t.Helper()
	if err := g.Deal(host.ID); err != nil {
		t.Fatal(err)
	}
}

func fakeIndex(t *testing.T, g *game.Game, players []game.Identity) int {
	t.Helper()
	fake := -1
	for i, p := range players {
		if viewOf(t, g, p).Round.IsFake {
			if fake != -1 {
				t.Fatal("more than one fake artist")
			}
			fake = i
		}
	}
	if fake == -1 {
		t.Fatal("no fake artist")
	}
	return fake
}

func TestDealGivesOneFakeAndOneSharedWord(t *testing.T) {
	g, players, _ := table(t, 6, testWords)
	for range 50 {
		deal(t, g, players[0])
		fake := fakeIndex(t, g, players)

		var word string
		for i, p := range players {
			round := viewOf(t, g, p).Round
			if i == fake {
				continue
			}
			if round.Word == "" {
				t.Fatal("artist has no word")
			}
			if word != "" && round.Word != word {
				t.Fatalf("artists have different words: %q, %q", word, round.Word)
			}
			word = round.Word
		}
	}
}

func TestDealSharesTheCategoryAndFirstPlayer(t *testing.T) {
	g, players, _ := table(t, 5, testWords)
	deal(t, g, players[0])

	first := viewOf(t, g, players[0]).Round
	if first.Category == "" || first.FirstPlayer == "" {
		t.Fatalf("got %+v", first)
	}
	for _, p := range players[1:] {
		round := viewOf(t, g, p).Round
		if round.Category != first.Category || round.FirstPlayer != first.FirstPlayer {
			t.Errorf("got %+v, want the same as %+v", round, first)
		}
	}
}

func TestWordMatchesItsCategory(t *testing.T) {
	g, players, _ := table(t, 4, testWords)
	categories := map[string]string{}
	for _, w := range testWords {
		categories[w.Word] = w.Category
	}
	for range 20 {
		deal(t, g, players[0])
		fake := fakeIndex(t, g, players)
		round := viewOf(t, g, players[(fake+1)%len(players)]).Round
		if categories[round.Word] != round.Category {
			t.Errorf("%q is not in category %q", round.Word, round.Category)
		}
	}
}

func TestFakeAndFirstPlayerAreFairAndIndependent(t *testing.T) {
	const size, deals = 10, 5000
	g, players, _ := table(t, size, testWords)
	names := map[string]int{}
	for i, p := range players {
		names[viewOf(t, g, p).You.Name] = i
	}

	fakeCounts := make([]int, size)
	firstCounts := make([]int, size)
	same := 0
	for range deals {
		deal(t, g, players[0])
		fake := fakeIndex(t, g, players)
		first := names[viewOf(t, g, players[fake]).Round.FirstPlayer]
		fakeCounts[fake]++
		firstCounts[first]++
		if fake == first {
			same++
		}
	}

	const expected, tolerance = deals / size, deals / size / 5
	for i := range size {
		if diff := fakeCounts[i] - expected; diff > tolerance || diff < -tolerance {
			t.Errorf("player %d was the fake artist %d times, want about %d", i, fakeCounts[i], expected)
		}
		if diff := firstCounts[i] - expected; diff > tolerance || diff < -tolerance {
			t.Errorf("player %d went first %d times, want about %d", i, firstCounts[i], expected)
		}
	}
	if diff := same - expected; diff > tolerance || diff < -tolerance {
		t.Errorf("the fake artist went first %d times, want about %d", same, expected)
	}
}

func TestScriptedDealPicksTheGivenPlayers(t *testing.T) {
	g, players, rng := table(t, 5, testWords)
	rng.Push(2, 3)
	deal(t, g, players[0])
	if got := fakeIndex(t, g, players); got != 2 {
		t.Errorf("got fake %d, want 2", got)
	}
	if got := viewOf(t, g, players[0]).Round.FirstPlayer; got != "Player3" {
		t.Errorf("got first player %q", got)
	}
}

func TestDealRequiresEnoughPlayers(t *testing.T) {
	g, players, _ := table(t, 3, testWords)
	wantErr(t, g.Deal(players[0].ID), game.ErrWrongPlayerCount)

	alex, err := g.Join("Alex")
	if err != nil {
		t.Fatal(err)
	}
	deal(t, g, players[0])
	if viewOf(t, g, alex).Status != game.StatusDealt {
		t.Error("round did not start with 4 players")
	}
}

func TestOnlyTheHostCanDeal(t *testing.T) {
	g, players, _ := table(t, 4, testWords)
	wantErr(t, g.Deal(players[1].ID), game.ErrNotHost)
}

func TestDeckHasNoRepeatsUntilUsedUp(t *testing.T) {
	var words []game.WordEntry
	for i := range 30 {
		words = append(words, game.WordEntry{Category: "Test", Word: fmt.Sprintf("Word%d", i)})
	}
	g, players, _ := table(t, 4, words)

	word := func() string {
		deal(t, g, players[0])
		fake := fakeIndex(t, g, players)
		return viewOf(t, g, players[(fake+1)%len(players)]).Round.Word
	}
	var order []string
	seen := map[string]bool{}
	for range words {
		w := word()
		if seen[w] {
			t.Fatalf("%q repeated before the list was used up", w)
		}
		seen[w] = true
		order = append(order, w)
	}
	for i, want := range order {
		if got := word(); got != want {
			t.Fatalf("after wrapping, word %d is %q, want the oldest word %q", i, got, want)
		}
	}
}

func TestRevealShowsTheAnswerToEveryone(t *testing.T) {
	g, players, rng := table(t, 5, testWords)
	rng.Push(1, 2)
	deal(t, g, players[0])
	word := viewOf(t, g, players[0]).Round.Word

	if err := g.Reveal(players[0].ID); err != nil {
		t.Fatal(err)
	}
	for _, p := range players {
		view := viewOf(t, g, p)
		if view.Status != game.StatusRevealed {
			t.Errorf("got status %q", view.Status)
		}
		if view.Round.FakeName != "Player1" || view.Round.Word != word {
			t.Errorf("got %+v", view.Round)
		}
	}
}

func TestRevealStillWorksAfterPlayersLeave(t *testing.T) {
	g, players, rng := table(t, 5, testWords)
	rng.Push(1, 2)
	deal(t, g, players[0])
	for _, p := range players[1:3] {
		if err := g.Remove(players[0].ID, p.ID); err != nil {
			t.Fatal(err)
		}
	}

	if got := viewOf(t, g, players[0]).Round.FirstPlayer; got != "Player2" {
		t.Errorf("got first player %q after they left", got)
	}
	if err := g.Reveal(players[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := viewOf(t, g, players[0]).Round.FakeName; got != "Player1" {
		t.Errorf("got fake artist %q after they left", got)
	}
}

func TestRevealRules(t *testing.T) {
	g, players, _ := table(t, 4, testWords)
	wantErr(t, g.Reveal(players[0].ID), game.ErrNoRound)

	deal(t, g, players[0])
	wantErr(t, g.Reveal(players[1].ID), game.ErrNotHost)
}

func TestDealAfterRevealStartsAFreshRound(t *testing.T) {
	g, players, _ := table(t, 4, testWords)
	deal(t, g, players[0])
	if err := g.Reveal(players[0].ID); err != nil {
		t.Fatal(err)
	}
	deal(t, g, players[0])

	view := viewOf(t, g, players[0])
	if view.Status != game.StatusDealt || view.Round.FakeName != "" || view.Round.Number != 2 {
		t.Errorf("got %+v", view.Round)
	}
}
