package authn

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// tickingClock is a clock that moves forward a millisecond at each reading, so
// that the session times compare the same way however fast the test runs.
type tickingClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *tickingClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(time.Millisecond)
	return c.t
}

func (c *tickingClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// newTestSessionManager is a session manager without the watcher's goroutine
// (the tests call watch themselves) and with a tickingClock.
func newTestSessionManager() *SessionManager {
	sm := newSessionManager(nil, true, nil)
	clock := &tickingClock{t: time.Unix(1_000_000, 0)}
	sm.now = clock.now
	return sm
}

func checkStats(t *testing.T, sm *SessionManager, expected SessionStatistics) {
	t.Helper()
	stats := sm.Statistics()
	if stats != expected {
		t.Errorf("expected stats %+v, got %+v", expected, stats)
	}
}

// watch() acquired the mutex and, when paused, returned without releasing it,
// permanently deadlocking the session manager. This asserts a paused watch()
// leaves the lock free for the next acquirer.
func TestWatchDoesNotLeakLockWhenPaused(t *testing.T) {
	sm := &SessionManager{slots: map[string]*SessionList{}}
	sm.pauseWatcher(true)
	sm.watch(time.Second)

	done := make(chan struct{})
	go func() {
		sm.Statistics() // acquires sm.mtx; blocks forever if watch() leaked the lock
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch() leaked the mutex while paused")
	}
}

func TestSessionManager(t *testing.T) {

	sm := newTestSessionManager()

	checkStats(t, sm, SessionStatistics{0, 0, 0})

	s11, n := sm.getSession("1")
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})
	if !n {
		t.Error("expected new session")
	}

	s12, n := sm.getSession("1")
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{2, 2, 1})
	if !n {
		t.Error("expected new session")
	}

	s21, n := sm.getSession("2")
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{3, 3, 2})
	if !n {
		t.Error("expected new session")
	}

	sm.leaveSession(s11)
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{2, 2, 2})

	s13, n := sm.getSession("1")
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{3, 3, 2})
	if !n {
		t.Error("expected new session")
	}

	sm.leaveSession(s12)
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{2, 2, 2})

	sm.leaveSession(s21)
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})

	sm.leaveSession(s13)
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
}

func TestSessionReuse(t *testing.T) {
	sm := newTestSessionManager()

	// Create a session and set Claims on it
	s1, isNew := sm.getSession("key1")
	if !isNew {
		t.Fatal("expected new session")
	}
	s1.Claims = &Claims{Role: "user1", RawClaims: `{"role":"user1"}`}

	// Leave the session — it stays in the pool
	sm.leaveSession(s1)

	// Re-acquire same key with a long timeout so it's not expired
	s2, isNew := sm.getSession("key1")
	if isNew {
		t.Fatal("expected reused session, got new")
	}
	if s2 != s1 {
		t.Fatal("expected same session object on reuse")
	}
	if s2.Claims == nil || s2.Claims.Role != "user1" {
		t.Error("Claims lost on session reuse")
	}
	if s2.Claims.RawClaims != `{"role":"user1"}` {
		t.Error("RawClaims lost on session reuse — this would cause empty jwt.claims in PostgreSQL")
	}

	sm.leaveSession(s2)
}

func TestSessionWatcherReleasesIdleSession(t *testing.T) {
	sm := newTestSessionManager()

	s, _ := sm.getSession("k")
	sm.leaveSession(s)

	// With zero timeouts, watch should remove the idle session
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
}

func TestSessionWatcherPreservesInUseSessions(t *testing.T) {
	sm := newTestSessionManager()

	s, _ := sm.getSession("k")
	// Don't leave — session is in use

	// Watcher should NOT remove in-use sessions
	sm.watch(0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})

	sm.leaveSession(s)
}

func TestSessionNotReusedWhileInUse(t *testing.T) {
	sm := newTestSessionManager()

	// Get session for key "k" — it's in use
	s1, isNew := sm.getSession("k")
	if !isNew {
		t.Fatal("expected new")
	}

	// Get another session for same key — s1 is still in use, so we get a new one
	s2, isNew := sm.getSession("k")
	if !isNew {
		t.Fatal("expected new session because first is still in use")
	}
	if s1 == s2 {
		t.Fatal("should not reuse in-use session")
	}

	sm.leaveSession(s1)
	sm.leaveSession(s2)
}

func TestSessionDisabledManager(t *testing.T) {
	sm := NewSessionManager(nil, false, nil)

	s1, isNew := sm.getSession("k")
	if !isNew {
		t.Fatal("disabled manager should always return new")
	}
	s1.Claims = &Claims{Role: "test"}

	ok := sm.leaveSession(s1)
	if !ok {
		t.Fatal("leave should return true for disabled manager")
	}

	// Should always create new, never reuse
	s2, isNew := sm.getSession("k")
	if !isNew {
		t.Fatal("disabled manager should always return new")
	}
	if s2 == s1 {
		t.Fatal("disabled manager should not reuse sessions")
	}
}

func TestSessionConcurrentAccess(t *testing.T) {
	sm := newTestSessionManager()
	const goroutines = 20
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			k := fmt.Sprintf("user%d", id%5) // 5 distinct keys, contention
			for i := 0; i < iterations; i++ {
				s, _ := sm.getSession(k)
				s.Claims = &Claims{Role: k}
				sm.leaveSession(s)
			}
		}(g)
	}
	wg.Wait()

	// After all goroutines done, stats should be consistent
	stats := sm.Statistics()
	if stats.InUse != 0 {
		t.Errorf("expected 0 in-use sessions after all goroutines done, got %d", stats.InUse)
	}
}

func TestSessionConcurrentWithWatcher(t *testing.T) {
	sm := newTestSessionManager()
	const goroutines = 10
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines + 1)

	// Watcher goroutine running concurrently
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				sm.watch(5 * time.Second)
				time.Sleep(time.Millisecond)
			}
		}
	}()

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			k := fmt.Sprintf("user%d", id%3)
			for i := 0; i < iterations; i++ {
				s, _ := sm.getSession(k)
				s.Claims = &Claims{Role: k}
				// Simulate some work
				time.Sleep(10 * time.Microsecond)
				sm.leaveSession(s)
			}
		}(g)
	}

	// Wait for workers, then stop watcher
	time.Sleep(500 * time.Millisecond)
	close(done)
	wg.Wait()

	stats := sm.Statistics()
	if stats.InUse != 0 {
		t.Errorf("expected 0 in-use after completion, got %d", stats.InUse)
	}
}

func BenchmarkSessionGetLeave(b *testing.B) {
	sm := NewSessionManager(nil, true, nil)
	for i := 0; i < b.N; i++ {
		s, _ := sm.getSession("bench-key")
		sm.leaveSession(s)
	}
}

func BenchmarkSessionGetLeaveParallel(b *testing.B) {
	sm := NewSessionManager(nil, true, nil)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s, _ := sm.getSession("bench-key")
			sm.leaveSession(s)
		}
	})
}

func BenchmarkSessionMultiKey(b *testing.B) {
	sm := NewSessionManager(nil, true, nil)
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s, _ := sm.getSession(keys[i%len(keys)])
			sm.leaveSession(s)
			i++
		}
	})
}

// BenchmarkSessionReuse measures the reuse path: serial get/leave on the same key
// so every iteration after the first reuses the existing session.
func BenchmarkSessionReuse(b *testing.B) {
	sm := NewSessionManager(nil, true, nil)
	// Warm up: create and leave one session so it's available for reuse
	s, _ := sm.getSession("reuse-key")
	s.Claims = &Claims{Role: "user", RawClaims: `{"role":"user"}`}
	sm.leaveSession(s)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, _ := sm.getSession("reuse-key")
		sm.leaveSession(s)
	}
}

// BenchmarkSessionParallelPerGoroutine gives each goroutine its own key,
// so there's no cross-goroutine contention — pure lock overhead.
func BenchmarkSessionParallelPerGoroutine(b *testing.B) {
	sm := NewSessionManager(nil, true, nil)
	var id atomic.Int64
	b.RunParallel(func(pb *testing.PB) {
		k := fmt.Sprintf("g%d", id.Add(1))
		for pb.Next() {
			s, _ := sm.getSession(k)
			sm.leaveSession(s)
		}
	})
}

// BenchmarkSessionWithWatcher runs get/leave with the watcher active,
// matching production conditions.
func BenchmarkSessionWithWatcher(b *testing.B) {
	shutdown := make(chan struct{})
	sm := NewSessionManager(nil, true, shutdown)
	defer close(shutdown)

	keys := []string{"w1", "w2", "w3", "w4", "w5"}
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s, _ := sm.getSession(keys[i%len(keys)])
			sm.leaveSession(s)
			i++
		}
	})
}

// A session is created before the token is verified. When verification (or the
// database lookup) fails, the half-built session must not stay in the manager:
// the next request with the same token would find it, skip verification, and
// run with nil claims or a nil database.
func TestDropSessionRemovesAHalfBuiltSession(t *testing.T) {
	sm := newTestSessionManager()
	s, isNew := sm.getSession("k")
	if !isNew {
		t.Fatal("expected a new session")
	}
	sm.dropSession(s)
	sm.watch(time.Hour)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
	if _, isNew := sm.getSession("k"); !isNew {
		t.Fatal("a dropped session must not be found again")
	}
}

// The session map grows by one entry per distinct token; a scan of random
// bearer tokens would grow it without bound. Beyond MaxSessions the manager
// hands out transient sessions that are never stored, and leaving or dropping
// one is a no-op instead of a nil-list panic.
func TestSessionCapHandsOutTransientSessions(t *testing.T) {
	sm := newTestSessionManager()
	sm.SetMaxSessions(2)
	a, _ := sm.getSession("a")
	b, _ := sm.getSession("b")
	c, isNew := sm.getSession("c")
	if !isNew || !c.transient {
		t.Fatalf("expected a transient session beyond the cap, got new=%v transient=%v", isNew, c.transient)
	}
	if a.transient || b.transient {
		t.Fatal("sessions within the cap must be stored")
	}
	sm.leaveSession(c)
	sm.dropSession(c)
	sm.watch(time.Hour)
	checkStats(t, sm, SessionStatistics{2, 2, 2})
	// once a stored session is gone, the slot is available again
	sm.dropSession(a)
	if d, _ := sm.getSession("d"); d.transient {
		t.Fatal("expected a stored session once below the cap")
	}
}

// A session hit skips token verification, so the one thing verification would
// have caught meanwhile must be checked by hand: the expiry, with the same 30 s
// clock skew PostgREST allows (Auth/Jwt.hs:70-76, checkForErrors: a token is
// expired when now - allowedSkewSeconds > exp).
func TestClaimsExpired(t *testing.T) {
	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	if claimsExpired(&Claims{}, now) {
		t.Error("claims without exp never expire")
	}
	cases := []struct {
		exp     time.Duration
		expired bool
	}{
		{-31 * time.Second, true},
		{-30 * time.Second, false}, // now - 30 > exp is false at the edge
		{-time.Second, false},
		{0, false},
		{time.Second, false},
	}
	for _, c := range cases {
		claims := &Claims{}
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(c.exp))
		if got := claimsExpired(claims, now); got != c.expired {
			t.Errorf("exp = now %+v: expired %v, want %v", c.exp, got, c.expired)
		}
	}
	// the clock is read in whole seconds, as PostgREST floors it
	claims := &Claims{}
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(-30 * time.Second))
	if claimsExpired(claims, now.Add(999*time.Millisecond)) {
		t.Error("a fraction of a second past the edge is still within the skew")
	}
}

// On shutdown the watcher emptied the session map without the manager's
// lock while requests used it. A request in flight leaves its session after
// the shutdown, and the requests after it get sessions that are not stored.
// Run with -race.
func TestShutdownLeavesTheRequestsInFlightAlone(t *testing.T) {
	shutdown := make(chan struct{})
	sm := newSessionManager(nil, true, shutdown)
	inFlight, _ := sm.getSession("busy")
	idle, _ := sm.getSession("idle")
	sm.leaveSession(idle)

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				s, _ := sm.getSession(fmt.Sprint("k", i%3))
				sm.leaveSession(s)
			}
		}(i)
	}
	watcher := make(chan struct{})
	go func() {
		sm.sessionWatcher()
		close(watcher)
	}()
	time.Sleep(20 * time.Millisecond)
	close(shutdown)
	<-watcher
	close(stop)
	wg.Wait()

	sm.leaveSession(inFlight)
	if _, isNew := sm.getSession("idle"); !isNew {
		t.Error("a session stored before the shutdown was found after it")
	}
	if total := sm.total; total != 0 {
		t.Errorf("%d sessions stored after the shutdown", total)
	}
}
