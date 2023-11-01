package server

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"
)

type Session struct {
	Claims     *Claims
	LastUsedAt time.Time
	Db         *database.Database
	DbConn     *database.DbPoolConn
	key        string
	inUse      atomic.Bool
	prev       *Session
	next       *Session
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

type SessionManager struct {
	slots   map[string]*SessionList
	mtx     sync.Mutex
	logger  *logging.Logger
	enabled bool
}

func (s *Server) initSessionManager() {
	s.sessionManager = &SessionManager{
		slots:   map[string]*SessionList{},
		logger:  s.Logger,
		enabled: s.Config.SessionMode != "none",
	}
	if s.sessionManager.enabled {
		go sessionWatcher(s)
	}
}

func sessionWatcher(s *Server) {
	sm := s.sessionManager
	ticker := time.NewTicker(1000 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			sm.mtx.Lock()
			var count, inUse int

			for k, list := range sm.slots {

				for session := list.head; session != nil; session = session.next {

					count += 1

					if session.inUse.Load() {
						inUse += 1
						continue
					}
					// Here we have a session not in use, which cannot be used now
					// because we hold a lock on the session manager and making
					// the session usable requires a lock

					spentTime := now.Sub(session.LastUsedAt)

					if spentTime > 5*time.Second {

						// Delete the session
						list.remove(session)
						if session.prev == nil {
							session = list.head
							if session == nil {
								break
							}
						} else {
							session = session.prev
						}

					} else if spentTime > 5*time.Second && session.DbConn != nil {

						// Release and detach the database connection from the session
						// (Acquire and attach are done in the auth middleware)
						err := database.ReleaseConnection(context.Background(), session.DbConn, false, true)
						if err != nil {
							sm.logger.Err(err).Msg("error releasing an expired session")
						}
						session.DbConn = nil
					}
				}
				if list.isEmpty() {
					delete(sm.slots, k)
				}
			}
			//fmt.Println("sessions: ", count, " in use: ", inUse)
			sm.mtx.Unlock()

		case <-s.shutdown:
			for k, list := range sm.slots {
				for session := list.head; session != nil; session = session.next {
					if session.DbConn != nil {
						database.ReleaseConnection(context.Background(), session.DbConn, false, false)
					}
					delete(sm.slots, k)
				}
			}
			return
		}
	}
}

func (sm *SessionManager) newSession(key string) *Session {
	now := time.Now()
	session := &Session{key: key}
	session.inUse.Store(true)
	session.LastUsedAt = now
	list := sm.slots[key]
	if list == nil {
		list = &SessionList{}
		sm.slots[key] = list
	}
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
	if list == nil || list.isEmpty() {
		return sm.newSession(key), true
	}
	session := list.head
	swapped := session.inUse.CompareAndSwap(false, true)
	if !swapped {
		return sm.newSession(key), true
	}
	list.frontToBack()
	return session, false
}

func (sm *SessionManager) leaveSession(session *Session) bool {
	if !sm.enabled {
		return true
	}
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	list := sm.slots[session.key]
	list.toFront(session)
	session.LastUsedAt = time.Now()
	swapped := session.inUse.CompareAndSwap(true, false)
	return swapped
}
