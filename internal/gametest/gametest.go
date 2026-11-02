// Package gametest provides a controllable clock and random source for tests.
package gametest

import (
	"math/rand/v2"
	"sort"
	"sync"
	"time"

	"fakeartist/internal/game"
)

// Clock is a fake game.Clock whose time moves only when Advance is called.
type Clock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*timer
}

type timer struct {
	clock *Clock
	at    time.Time
	f     func()
}

// NewClock returns a Clock set to a fixed time.
func NewClock() *Clock {
	return &Clock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) AfterFunc(d time.Duration, f func()) game.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &timer{clock: c, at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	return t
}

// Pending returns the number of scheduled timers.
func (c *Clock) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}

// Advance moves time forward, running each timer that comes due, in order.
func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	target := c.now.Add(d)
	for {
		sort.SliceStable(c.timers, func(i, j int) bool { return c.timers[i].at.Before(c.timers[j].at) })
		if len(c.timers) == 0 || c.timers[0].at.After(target) {
			break
		}
		next := c.timers[0]
		c.timers = c.timers[1:]
		c.now = next.at
		c.mu.Unlock()
		next.f()
		c.mu.Lock()
	}
	c.now = target
	c.mu.Unlock()
}

func (t *timer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	for i, other := range t.clock.timers {
		if other == t {
			t.clock.timers = append(t.clock.timers[:i], t.clock.timers[i+1:]...)
			return true
		}
	}
	return false
}

// Rand is a seeded game.Rand. Values given to Push are returned first.
type Rand struct {
	mu     sync.Mutex
	r      *rand.Rand
	script []int
}

// NewRand returns a Rand with a fixed seed.
func NewRand(seed uint64) *Rand {
	return &Rand{r: rand.New(rand.NewPCG(seed, seed))}
}

// Push queues values for upcoming IntN calls.
func (r *Rand) Push(values ...int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.script = append(r.script, values...)
}

func (r *Rand) IntN(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.script) > 0 {
		v := r.script[0]
		r.script = r.script[1:]
		return v % n
	}
	return r.r.IntN(n)
}

func (r *Rand) Token() string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	r.mu.Lock()
	defer r.mu.Unlock()
	token := make([]byte, 26)
	for i := range token {
		token[i] = alphabet[r.r.IntN(len(alphabet))]
	}
	return string(token)
}
