package authn

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func checkStats(t *testing.T, sm *SessionManager, expected SessionStatistics) {
	t.Helper()
	stats := sm.Statistics()
	if stats != expected {
		t.Errorf("expected stats %+v, got %+v", expected, stats)
	}
}

func TestSessionManager(t *testing.T) {

	sm := NewSessionManager(nil, true, nil)

	checkStats(t, sm, SessionStatistics{0, 0, 0})

	s11, n := sm.getSession("1")
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})
	if !n {
		t.Error("expected new session")
	}

	s12, n := sm.getSession("1")
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{2, 2, 1})
	if !n {
		t.Error("expected new session")
	}

	s21, n := sm.getSession("2")
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{3, 3, 2})
	if !n {
		t.Error("expected new session")
	}

	sm.leaveSession(s11)
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{2, 2, 2})

	s13, n := sm.getSession("1")
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{3, 3, 2})
	if !n {
		t.Error("expected new session")
	}

	sm.leaveSession(s12)
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{2, 2, 2})

	sm.leaveSession(s21)
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})

	sm.leaveSession(s13)
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
}

func TestSessionReuse(t *testing.T) {
	sm := NewSessionManager(nil, true, nil)

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
	sm := NewSessionManager(nil, true, nil)

	s, _ := sm.getSession("k")
	sm.leaveSession(s)

	// With zero timeouts, watch should remove the idle session
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
}

func TestSessionWatcherPreservesInUseSessions(t *testing.T) {
	sm := NewSessionManager(nil, true, nil)

	s, _ := sm.getSession("k")
	// Don't leave — session is in use

	// Watcher should NOT remove in-use sessions
	sm.watch(0, 0)
	checkStats(t, sm, SessionStatistics{1, 1, 1})

	sm.leaveSession(s)
}

func TestSessionWatcherConnTimeout(t *testing.T) {
	sm := NewSessionManager(nil, true, nil)

	s, _ := sm.getSession("k")
	// Session has no DbConn — simulate that it was never acquired
	sm.leaveSession(s)

	// With a long conn timeout but zero session timeout, session is deleted
	// (because DbConn is nil and sessionTimeout is 0)
	sm.watch(0, time.Hour)
	checkStats(t, sm, SessionStatistics{0, 0, 0})
}

func TestSessionNotReusedWhileInUse(t *testing.T) {
	sm := NewSessionManager(nil, true, nil)

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
	sm := NewSessionManager(nil, true, nil)
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
	sm := NewSessionManager(nil, true, nil)
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
				sm.watch(5*time.Second, 1*time.Second)
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
