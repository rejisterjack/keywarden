package limit

import (
	"sync"
	"time"
)

// Limiter is a per-key token bucket.
type Limiter struct {
	mu      sync.Mutex
	rate    float64
	burst   float64
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	at     time.Time
}

// New returns a limiter that refills rate tokens per second, up to burst.
func New(rate, burst float64) *Limiter {
	return &Limiter{rate: rate, burst: burst, buckets: map[string]*bucket{}}
}

// Allow reports whether key may proceed now.
func (l *Limiter) Allow(key string) bool {
	if l == nil || key == "" {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.buckets) > 10000 {
		l.buckets = map[string]*bucket{}
	}
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: l.burst, at: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.at).Seconds()
		if elapsed > 0 {
			b.tokens += elapsed * l.rate
			if b.tokens > l.burst {
				b.tokens = l.burst
			}
			b.at = now
		}
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
