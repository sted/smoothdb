package api

import (
	"math"
	"net"
	"sync"
	"time"
)

// ipLimiter is a per-client token bucket used on the unauthenticated /token
// route. Each client gets perMinute tokens, refilled continuously at
// perMinute per minute; a request costs one token. Buckets idle for longer
// than idleTTL are dropped so a scan of many source addresses cannot grow the
// map without bound.
type ipLimiter struct {
	mu        sync.Mutex
	perMinute int
	now       func() time.Time
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	tokens float64
	seen   time.Time
}

const limiterIdleTTL = 10 * time.Minute

func newIPLimiter(perMinute int, now func() time.Time) *ipLimiter {
	return &ipLimiter{perMinute: perMinute, now: now, buckets: map[string]*bucket{}, lastSweep: now()}
}

// Allow consumes one token for the client. When the bucket is empty it
// returns false and the time until the next token is available.
func (l *ipLimiter) Allow(client string) (bool, time.Duration) {
	if l.perMinute <= 0 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)
	refillPerSecond := float64(l.perMinute) / 60
	b, ok := l.buckets[client]
	if !ok {
		b = &bucket{tokens: float64(l.perMinute), seen: now}
		l.buckets[client] = b
	} else {
		b.tokens = math.Min(float64(l.perMinute), b.tokens+now.Sub(b.seen).Seconds()*refillPerSecond)
		b.seen = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / refillPerSecond * float64(time.Second))
	return false, wait
}

// sweep drops buckets not seen for idleTTL; it runs at most once per idleTTL.
func (l *ipLimiter) sweep(now time.Time) {
	if now.Sub(l.lastSweep) < limiterIdleTTL {
		return
	}
	l.lastSweep = now
	for client, b := range l.buckets {
		if now.Sub(b.seen) >= limiterIdleTTL {
			delete(l.buckets, client)
		}
	}
}

// size reports the number of tracked clients (for tests)
func (l *ipLimiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

// clientIP extracts the client address from RemoteAddr. Proxy headers are
// deliberately not trusted here: an attacker controls them.
func clientIP(remoteAddr string) string {
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return host
	}
	return remoteAddr
}
