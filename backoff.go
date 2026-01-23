package jobq

import (
	"math"
	"math/rand"
	"sync/atomic"
	"time"
)

// Backoff is a time.Duration counter, starting at Min. After every call to
// the Duration method, the current timing is multiplied by Factor, but it
// never exceeds Max.
//
// Backoff is not generally concurrent-safe, but ForAttempt method can be used
// concurrently.
type Backoff struct {
	attempt atomic.Uint64
	Factor  float64       // Factor is the multiplying factor for each incrementing step; defaults to 2.
	Jitter  bool          // Jitter eases contention by randomizing backoff steps; defaults to false.
	Min     time.Duration // Min is the minimum value of counter; defaults to 100ms.
	Max     time.Duration // Max is the maximum value of counter; defaults to 10s.
}

// Duration returns the duration for the current attempt before incrementing the
// attempt counter. See ForAttempt.
func (b *Backoff) Duration() time.Duration {
	return b.ForAttempt(float64(b.attempt.Add(1) - 1))
}

const maxInt64 = float64(math.MaxInt64 - 512)

// ForAttempt returns the duration for a specific attempt. This is useful if you
// have a large number of independent Backoffs, but don't want to use unecessary
// memory storing the Backoff parameters per Backoff. The first attempt should be
// zero.
//
//	ForAttempt is concurrent safe.
func (b *Backoff) ForAttempt(attempt float64) time.Duration {
	minm := b.Min
	if minm <= 0 {
		minm = 100 * time.Millisecond
	}
	maxm := b.Max
	if maxm <= 0 {
		maxm = 10 * time.Second
	}
	if minm >= maxm {
		return maxm // short circuit
	}
	factor := b.Factor
	if factor <= 0 {
		factor = 2
	}
	minf := float64(minm)
	durf := minf * math.Pow(factor, attempt)

	if b.Jitter {
		durf = rand.Float64()*(durf-minf) + minf
	}
	// ensure float64 won't overflow int64
	if durf > maxInt64 {
		return maxm
	}
	dur := time.Duration(durf)
	// keep within bounds
	if dur < minm {
		return minm
	}
	if dur > maxm {
		return maxm
	}
	return dur
}

// Reset restarts the current attempt counter at zero.
func (b *Backoff) Reset() {
	b.attempt.Store(0)
}

// Attempt returns the current attempt counter-value.
func (b *Backoff) Attempt() float64 {
	return float64(b.attempt.Load())
}

// Copy returns the Backoff with equal constraints as the original.
func (b *Backoff) Copy() *Backoff {
	return &Backoff{
		Factor: b.Factor,
		Jitter: b.Jitter,
		Min:    b.Min,
		Max:    b.Max,
	}
}
