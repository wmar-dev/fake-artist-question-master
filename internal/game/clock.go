package game

import (
	"crypto/rand"
	"math/big"
	"time"
)

// Timer is a pending call scheduled with a Clock.
type Timer interface {
	Stop() bool
}

// Clock is the source of time, injected so tests can control it.
type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

// Rand is the source of randomness, injected so tests are deterministic.
type Rand interface {
	// IntN returns a uniform random integer in [0, n).
	IntN(n int) int
	// Token returns an unguessable string with at least 128 bits of entropy.
	Token() string
}

// RealClock is the Clock used outside tests.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }

func (RealClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// CryptoRand is the Rand used outside tests.
type CryptoRand struct{}

func (CryptoRand) IntN(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // crypto/rand does not fail on supported platforms.
	}
	return int(v.Int64())
}

func (CryptoRand) Token() string { return rand.Text() }
