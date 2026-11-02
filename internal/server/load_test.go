package server_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	loadGames     = 50
	playersInGame = 10
	maxHeapBytes  = 100 << 20
	maxUpdateSize = 2 << 10
)

// TestFiftyGamesOfTenPlayers checks SC-003, SC-005, and the memory budget. The
// heap figure includes the test's own clients, so the server uses less.
func TestFiftyGamesOfTenPlayers(t *testing.T) {
	h := newHarness(t)
	var streams []*stream
	var hosts []*player
	for range loadGames {
		players := h.group(playersInGame)
		hosts = append(hosts, players[0])
		for _, p := range players {
			streams = append(streams, p.stream())
		}
	}

	var wg sync.WaitGroup
	var dealt sync.WaitGroup
	dealt.Add(len(streams))
	for _, s := range streams {
		wg.Add(1)
		go func() {
			defer wg.Done()
			done := false
			for e := range s.events {
				if !done && strings.Contains(e.data, `"status":"dealt"`) {
					done = true
					dealt.Done()
				}
			}
		}()
	}

	start := time.Now()
	for _, host := range hosts {
		dealRound(t, host)
	}
	finished := make(chan struct{})
	go func() {
		dealt.Wait()
		close(finished)
	}()
	select {
	case <-finished:
		t.Logf("%d players updated in %v", len(streams), time.Since(start))
	case <-time.After(updateWait):
		t.Fatalf("not all players were updated within %v", updateWait)
	}

	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	t.Logf("heap in use: %d MB", mem.HeapInuse>>20)
	if mem.HeapInuse > maxHeapBytes {
		t.Errorf("heap in use is %d MB, over the budget of %d MB", mem.HeapInuse>>20, maxHeapBytes>>20)
	}
}

func TestLargestUpdateIsWithinBudget(t *testing.T) {
	h := newHarness(t)
	host := h.host(strings.Repeat("h", 20))
	for i := 1; i < playersInGame; i++ {
		h.join(host.code, fmt.Sprintf("%020d", i))
	}
	dealRound(t, host)

	s := host.stream()
	e := s.next()
	t.Logf("update size: %d bytes", len(e.data))
	if len(e.data) > maxUpdateSize {
		t.Errorf("update is %d bytes, over the budget of %d", len(e.data), maxUpdateSize)
	}
}
