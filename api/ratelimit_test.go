package api

import (
	"testing"
	"time"
)

// /token is the one route reachable without credentials, so it gets a per-IP
// token bucket: a client may make perMinute attempts, then is refused with a
// Retry-After until the bucket refills; other clients are unaffected.
func TestIPLimiterBucketPerClient(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	l := newIPLimiter(3, func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("10.0.0.1"); !ok {
			t.Fatalf("attempt %d from 10.0.0.1 should be allowed", i+1)
		}
	}
	ok, retry := l.Allow("10.0.0.1")
	if ok {
		t.Fatal("4th attempt within the minute should be refused")
	}
	if retry <= 0 || retry > time.Minute {
		t.Errorf("expected a Retry-After within a minute, got %v", retry)
	}
	// another client has its own bucket
	if ok, _ := l.Allow("10.0.0.2"); !ok {
		t.Fatal("a different client must not be affected")
	}
	// after the refill interval one token is back
	now = now.Add(retry)
	if ok, _ := l.Allow("10.0.0.1"); !ok {
		t.Fatal("after Retry-After the client should be allowed again")
	}
}

// A non-positive limit disables the limiter entirely.
func TestIPLimiterDisabled(t *testing.T) {
	l := newIPLimiter(0, time.Now)
	for i := 0; i < 1000; i++ {
		if ok, _ := l.Allow("10.0.0.1"); !ok {
			t.Fatal("a disabled limiter must allow everything")
		}
	}
}

// Buckets of clients not seen for a while are dropped, so the map cannot grow
// without bound under a scan of many source addresses.
func TestIPLimiterForgetsIdleClients(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	l := newIPLimiter(3, func() time.Time { return now })
	for i := 0; i < 5000; i++ {
		l.Allow("10.0." + string(rune('a'+i%26)) + "." + string(rune('a'+i/26%26)))
	}
	now = now.Add(time.Hour)
	l.Allow("10.1.1.1")
	if n := l.size(); n > 1 {
		t.Errorf("expected idle buckets to be dropped, still tracking %d", n)
	}
}
