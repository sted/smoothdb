package authn

import (
	"context"
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
	// retainConnections keeps the prepared connection attached to the session
	// between requests ("role" mode); "claims" mode caches only the claims
	retainConnections bool
	// closed is set on shutdown: no session is stored any more
	closed bool
	// waiters are the requests waiting for a connection, per key, oldest
	// first: a session of the same key that leaves hands its own over (see
	// leaveSession). idle counts the sessions not in use that keep a
	// connection.
	waiters map[string][]*waiter
	// waiting counts the waiters per pool (database.PoolOf)
	waiting map[*database.Database]int
	idle    int
	// now is the clock of the session times, release gives a connection a
	// session kept back to its pool (the tests replace both)
	now     func() time.Time
	release func(conn *database.DbPoolConn)
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
	sm := &SessionManager{
		slots:             map[string]*SessionList{},
		logger:            logger,
		enabled:           enabled,
		shutdown:          shutdown,
		maxSessions:       DefaultMaxSessions,
		retainConnections: true,
		now:               time.Now,
	}
	sm.release = func(conn *database.DbPoolConn) {
		err := database.ReleaseConnection(context.Background(), conn, false, true)
		if err != nil && sm.logger != nil {
			sm.logger.Err(err).Msg("error releasing the connection of a session")
		}
	}
	return sm
}

func (sm *SessionManager) watch(sessionTimeout time.Duration, connTimeout time.Duration) {
	now := sm.clock()
	// the connections to give back: released once the lock is free, so that
	// no request waits for their round trips (each bounded, see
	// database.ReleaseConnection)
	var expired []*database.DbPoolConn
	defer func() {
		for _, conn := range expired {
			sm.release(conn)
		}
	}()
	sm.mtx.Lock()
	defer sm.mtx.Unlock()

	if sm.paused {
		return
	}
	sm.count = 0
	sm.inUse = 0
	sm.idle = 0

	for k, list := range sm.slots {

		for session := list.head; session != nil; session = session.next {

			sm.count += 1

			if session.inUse.Load() {
				sm.inUse += 1
				continue
			}
			// Here we have a session not in use, which cannot be used now
			// because we hold a lock on the session manager and making
			// the session usable requires a lock

			spentTime := now.Sub(session.LastUsedAt)

			if spentTime > sessionTimeout && session.DbConn == nil {

				// Delete the session
				list.remove(session)
				sm.count -= 1
				sm.total -= 1

			} else if spentTime > connTimeout && session.DbConn != nil {

				// Detach the database connection from the session, to release
				// (Acquire and attach are done in the auth middleware)
				// The session is preserved and can be reused
				expired = append(expired, session.DbConn)
				session.DbConn = nil
			}
			if session.DbConn != nil {
				sm.idle += 1 // recounted, as count and inUse
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
			sm.watch(5*time.Second, 1*time.Second)
		case <-sm.shutdown:
			sm.close()
			return
		}
	}
}

// close empties the manager on shutdown: the idle sessions give their
// connections back, a request in flight keeps its own until it leaves (see
// leaveSession), and the requests that come after get transient sessions.
func (sm *SessionManager) close() {
	var conns []*database.DbPoolConn
	sm.mtx.Lock()
	sm.closed = true
	for _, list := range sm.slots {
		for session := list.head; session != nil; session = session.next {
			if !session.inUse.Load() && session.DbConn != nil {
				conns = append(conns, session.DbConn)
				session.DbConn = nil
			}
		}
	}
	sm.slots = map[string]*SessionList{}
	sm.total = 0
	sm.idle = 0
	sm.mtx.Unlock()
	for _, conn := range conns {
		sm.release(conn)
	}
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
	if session.DbConn != nil {
		sm.idle -= 1
	}
	list.frontToBack()
	return session, false
}

func (sm *SessionManager) leaveSession(session *Session) bool {
	if !sm.enabled {
		return true
	}
	if session.transient {
		return true
	}
	sm.mtx.Lock()
	if sm.closed {
		// emptied by the shutdown while the request ran: nothing keeps the
		// session's connection any more
		conn := session.DbConn
		session.DbConn = nil
		sm.mtx.Unlock()
		if conn != nil {
			sm.release(conn)
		}
		return true
	}
	var yielded *database.DbPoolConn
	if conn := session.DbConn; conn != nil {
		own := sm.waiters[session.key]
		others := sm.waiting[database.PoolOf(session.Db)] - len(own)
		if others > 0 || database.Acquiring(session.Db) {
			// a request of another token waits for this pool, registered
			// with the manager (perhaps not on the pool yet) or not a
			// session's: the connection goes back to the pool, reset, where
			// the requests wait in turn, those of this token too
			yielded = conn
			session.DbConn = nil
		} else if len(own) > 0 {
			// only requests of the same token wait: the connection is theirs
			// already (same role, same claims), handed over as it is
			own[0].conn <- conn
			own[0].cancel()
			sm.popWaiter(session.key)
			session.DbConn = nil
		} else {
			sm.idle += 1
		}
	}
	list := sm.slots[session.key]
	list.toFront(session)
	session.LastUsedAt = sm.clock()
	swapped := session.inUse.CompareAndSwap(true, false)
	sm.mtx.Unlock()
	if yielded != nil {
		// off the request's path: its answer does not wait for the reset
		go sm.release(yielded)
	}
	return swapped
}

// waiter is a request waiting for a connection: the pool's, or the one a
// session of its key leaves (see takeConnection).
type waiter struct {
	conn   chan *database.DbPoolConn // buffered: a hand-over never blocks
	cancel context.CancelFunc        // stops the wait on the pool
	pool   *database.Database        // the pool it waits for
}

// takeConnection gives the request of session a connection kept by an idle
// session: one of its own key (prepared already), or, when the pool is
// exhausted, one of any key. Otherwise it queues w for the next connection a
// session of the key leaves, and answers nil: the request then waits on the
// pool and on w at once, and calls dropWaiter when either answers.
func (sm *SessionManager) takeConnection(session *Session, w *waiter, exhausted bool) *database.DbPoolConn {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	if sm.idle > 0 {
		if conn := sm.detachIdle(sm.slots[session.key], session.Db); conn != nil {
			return conn
		}
		if exhausted {
			for _, list := range sm.slots {
				if conn := sm.detachIdle(list, session.Db); conn != nil {
					return conn
				}
			}
		}
	}
	// registered under the same lock as the scan: a session that leaves from
	// now on sees the wait, before the request reaches the pool
	if sm.waiters == nil {
		sm.waiters = map[string][]*waiter{}
		sm.waiting = map[*database.Database]int{}
	}
	w.pool = database.PoolOf(session.Db)
	sm.waiters[session.key] = append(sm.waiters[session.key], w)
	sm.waiting[w.pool] += 1
	return nil
}

// detachIdle takes the connection of an idle session of list, if one keeps
// one from the pool of db: a connection is bound to its database. Must be
// called with the lock held.
func (sm *SessionManager) detachIdle(list *SessionList, db *database.Database) *database.DbPoolConn {
	if list == nil {
		return nil
	}
	for s := list.head; s != nil; s = s.next {
		if !s.inUse.Load() && s.DbConn != nil && database.SamePool(s.Db, db) {
			conn := s.DbConn
			s.DbConn = nil
			sm.idle -= 1
			return conn
		}
	}
	return nil
}

// dropWaiter takes w out of the queue of key, if it is still there.
func (sm *SessionManager) dropWaiter(key string, w *waiter) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	q := sm.waiters[key]
	for i, x := range q {
		if x == w {
			q = append(q[:i:i], q[i+1:]...)
			sm.waiting[w.pool] -= 1
			break
		}
	}
	if len(q) == 0 {
		delete(sm.waiters, key)
	} else {
		sm.waiters[key] = q
	}
}

// popWaiter removes the first waiter of key. Must be called with the lock held.
func (sm *SessionManager) popWaiter(key string) {
	q := sm.waiters[key]
	sm.waiting[q[0].pool] -= 1
	if len(q) == 1 {
		delete(sm.waiters, key)
	} else {
		sm.waiters[key] = q[1:]
	}
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
	conn := session.DbConn
	session.DbConn = nil
	sm.mtx.Unlock()
	// A connection the session kept between requests ("role" mode, see the
	// watcher) goes back to the pool now: once dropped, the session is out of
	// the watcher's sight and nothing else would release it.
	if conn != nil {
		sm.release(conn)
	}
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

// SetRetainConnections selects between "role" mode (true: the prepared
// connection stays attached to the session between requests) and "claims"
// mode (false: only the verified claims are cached, the connection goes back
// to the pool at the end of every request).
func (sm *SessionManager) SetRetainConnections(retain bool) {
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	sm.retainConnections = retain
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
