package authn

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

// A Session for SmoothDB is the way to cache information about
// a user and his connection to a database
type Session struct {
	Claims     *Claims
	LastUsedAt time.Time
	Db         *database.Database
	DbConn     *database.DbPoolConn
	key        string
	inUse      atomic.Bool
	// transient sessions are handed out beyond MaxSessions (or after a drop):
	// they are never stored, so leaving or dropping them is a no-op
	transient bool
	prev      *Session
	next      *Session
}

type SessionList struct {
	head *Session
	tail *Session
}

func (sl *SessionList) isEmpty() bool {
	return sl.head == nil
}

func (sl *SessionList) append(s *Session) {
	if !sl.isEmpty() {
		sl.tail.next = s
		s.prev = sl.tail
		sl.tail = s
	} else {
		sl.head = s
		sl.tail = s
	}
}

func (sl *SessionList) remove(s *Session) {
	if s == sl.head {
		sl.head = s.next
	} else {
		s.prev.next = s.next
	}
	if s == sl.tail {
		sl.tail = s.prev
	} else {
		s.next.prev = s.prev
	}
}

func (sl *SessionList) frontToBack() {
	if sl.head == nil || sl.head == sl.tail {
		return
	}
	s := sl.head
	sl.tail.next = s
	sl.head = s.next
	s.next = nil
	s.prev = sl.tail
	sl.tail = s
}

func (sl *SessionList) toFront(s *Session) {
	if sl.head == s {
		return
	}
	if s == sl.tail {
		sl.tail = s.prev
	} else {
		s.next.prev = s.prev
	}
	s.prev.next = s.next
	s.next = sl.head
	s.prev = nil
	sl.head.prev = s
	sl.head = s
}

// SessionManager manages user sessions and monitors them efficiently.
// When disabled, it simply returns Sessions without managing or monitoring.
type SessionManager struct {
	slots    map[string]*SessionList
	mtx      sync.Mutex
	logger   *logging.Logger
	enabled  bool
	shutdown chan struct{}
	paused   bool
	count    int
	inUse    int
	// total is the live number of stored sessions (count is refreshed by watch)
	total       int
	maxSessions int
	// closed is set on shutdown: no session is stored any more
	closed bool
	// now is the clock of the session times (the tests replace it)
	now func() time.Time
}

// DefaultMaxSessions bounds the session map when no MaxSessions is configured
const DefaultMaxSessions = 10000

func NewSessionManager(logger *logging.Logger, enabled bool, shutdown chan struct{}) *SessionManager {
	sm := newSessionManager(logger, enabled, shutdown)
	if enabled {
		go sm.sessionWatcher()
	}
	return sm
}

// newSessionManager is NewSessionManager without the watcher's goroutine.
func newSessionManager(logger *logging.Logger, enabled bool, shutdown chan struct{}) *SessionManager {
	return &SessionManager{
		slots:       map[string]*SessionList{},
		logger:      logger,
		enabled:     enabled,
		shutdown:    shutdown,
		maxSessions: DefaultMaxSessions,
		now:         time.Now,
	}
}

// watch deletes the sessions unused for sessionTimeout, and counts the others.
// A session caches the verified claims of its token; the connection of its
// request goes back to the pool at the end of the request.
func (sm *SessionManager) watch(sessionTimeout time.Duration) {
	now := sm.clock()
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	if sm.paused {
		return
	}
	sm.count = 0
	sm.inUse = 0

	for k, list := range sm.slots {
		for session := list.head; session != nil; session = session.next {
			sm.count += 1
			if session.inUse.Load() {
				sm.inUse += 1
				continue
			}
			if now.Sub(session.LastUsedAt) > sessionTimeout {
				list.remove(session)
				sm.count -= 1
				sm.total -= 1
			}
		}
		if list.isEmpty() {
			delete(sm.slots, k)
		}
	}
}

func (sm *SessionManager) sessionWatcher() {

	ticker := time.NewTicker(1000 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.watch(5 * time.Second)
		case <-sm.shutdown:
			sm.close()
			return
		}
	}
}

// close empties the manager on shutdown, under its lock: a request in flight
// leaves its session as usual (see leaveSession), and the requests that come
// after get transient sessions.
func (sm *SessionManager) close() {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.closed = true
	sm.slots = map[string]*SessionList{}
	sm.total = 0
}

func (sm *SessionManager) newSession(key string) *Session {
	now := sm.clock()
	session := &Session{key: key}
	session.inUse.Store(true)
	session.LastUsedAt = now
	list := sm.slots[key]
	if list == nil {
		list = &SessionList{}
		sm.slots[key] = list
	}
	sm.total += 1
	list.append(session)
	return session
}

func (sm *SessionManager) getSession(key string) (*Session, bool) {
	if !sm.enabled {
		return &Session{key: key}, true
	}
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	list := sm.slots[key]
	if list == nil || list.isEmpty() || sm.closed {
		return sm.newSessionOrTransient(key), true
	}
	session := list.head
	swapped := session.inUse.CompareAndSwap(false, true)
	if !swapped {
		return sm.newSessionOrTransient(key), true
	}
	list.frontToBack()
	return session, false
}

func (sm *SessionManager) leaveSession(session *Session) bool {
	if !sm.enabled || session.transient {
		return true
	}
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	if sm.closed {
		// emptied by the shutdown while the request ran
		return true
	}
	list := sm.slots[session.key]
	list.toFront(session)
	session.LastUsedAt = sm.clock()
	swapped := session.inUse.CompareAndSwap(true, false)
	return swapped
}

func (sm *SessionManager) pauseWatcher(pause bool) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.paused = pause
}

type SessionStatistics struct {
	Count int
	InUse int
	Users int
}

func (sm *SessionManager) Statistics() SessionStatistics {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	return SessionStatistics{sm.count, sm.inUse, len(sm.slots)}
}

// newSessionOrTransient stores a new session unless the manager is at its cap,
// in which case it hands out a transient one: the request runs as if sessions
// were disabled, and the map cannot grow without bound under a scan of tokens.
// Must be called with the lock held.
func (sm *SessionManager) newSessionOrTransient(key string) *Session {
	if sm.total >= sm.maxSessions || sm.closed {
		s := &Session{key: key, transient: true}
		s.inUse.Store(true)
		return s
	}
	return sm.newSession(key)
}

// dropSession removes a session from the manager for good. It is used for a
// half-built session (its request failed before claims or database were set)
// and for a session whose token has expired: the next request with the same
// token must build a new one and verify the token again.
func (sm *SessionManager) dropSession(session *Session) {
	if !sm.enabled || session.transient {
		return
	}
	sm.mtx.Lock()
	if list := sm.slots[session.key]; list != nil {
		list.remove(session)
		sm.total -= 1
		if list.isEmpty() {
			delete(sm.slots, session.key)
		}
	}
	session.transient = true // later leave/drop calls are no-ops
	sm.mtx.Unlock()
}

// SetMaxSessions bounds the number of stored sessions (non-positive: default)
func (sm *SessionManager) SetMaxSessions(n int) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	if n <= 0 {
		n = DefaultMaxSessions
	}
	sm.maxSessions = n
}

// claimsExpired reports whether cached claims are past their exp, with the
// same rule and 30 s skew as the verifier (PostgREST's checkForErrors: expired
// when now - 30 > exp, in whole seconds). A session hit skips token
// verification, so this is the one check that must be repeated.
func claimsExpired(c *Claims, now time.Time) bool {
	if c == nil || c.ExpiresAt == nil {
		return false
	}
	return now.Unix()-allowedSkew > c.ExpiresAt.Unix()
}

// clock is the time of the session manager (its now, time.Now by default).
func (sm *SessionManager) clock() time.Time {
	if sm.now == nil {
		return time.Now()
	}
	return sm.now()
}
