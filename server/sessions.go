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
	Key        string
	Claims     *Claims
	InUse      atomic.Bool
	LastUsedAt time.Time
	Db         *database.Database
	DbConn     *database.DbPoolConn
	Prev       *Session
	Next       *Session
}

type SessionList struct {
	Head *Session
	Tail *Session
}

func (sl *SessionList) isEmpty() bool {
	return sl.Head == nil
}

func (sl *SessionList) append(s *Session) {
	if !sl.isEmpty() {
		sl.Tail.Next = s
		s.Prev = sl.Tail
		sl.Tail = s
	} else {
		sl.Head = s
		sl.Tail = s
	}
}

func (sl *SessionList) remove(s *Session) {
	if s == sl.Head {
		sl.Head = s.Next
	} else {
		s.Prev.Next = s.Next
	}
	if s == sl.Tail {
		sl.Tail = s.Prev
	} else {
		s.Next.Prev = s.Prev
	}
}

func (sl *SessionList) frontToBack() {
	if sl.Head == nil || sl.Head == sl.Tail {
		return
	}
	s := sl.Head
	sl.Tail.Next = s
	sl.Head = s.Next
	s.Next = nil
	s.Prev = sl.Tail
	sl.Tail = s
}

func (sl *SessionList) toFront(s *Session) {
	if sl.Head == s {
		return
	}
	if s == sl.Tail {
		sl.Tail = s.Prev
	} else {
		s.Next.Prev = s.Prev
	}
	s.Prev.Next = s.Next
	s.Next = sl.Head
	s.Prev = nil
	sl.Head.Prev = s
	sl.Head = s
}

type SessionManager struct {
	Slots   map[string]*SessionList
	mtx     sync.Mutex
	logger  *logging.Logger
	enabled bool
}

func (s *Server) initSessionManager() {
	s.sessionManager = &SessionManager{
		Slots:   map[string]*SessionList{},
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

			for k, list := range sm.Slots {

				for session := list.Head; session != nil; session = session.Next {

					count += 1

					if session.InUse.Load() {
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
						if session.Prev == nil {
							session = list.Head
							if session == nil {
								break
							}
						} else {
							session = session.Prev
						}

					} else if spentTime > 1000*time.Millisecond && session.DbConn != nil {

						// Release and detach the database connection from the session
						// (Acquire and attach are done in the auth middleware)
						err := database.ReleaseConnection(context.Background(), session.DbConn, true)
						if err != nil {
							sm.logger.Err(err).Msg("error releasing an expired session")
						}
						session.DbConn = nil
					}
				}
				if list.isEmpty() {
					delete(sm.Slots, k)
				}
			}
			//fmt.Println("sessions: ", count, " in use: ", inUse)
			sm.mtx.Unlock()

		case <-s.shutdown:
			for k, list := range sm.Slots {
				for session := list.Head; session != nil; session = session.Next {
					if session.DbConn != nil {
						database.ReleaseConnection(context.Background(), session.DbConn, false)
					}
					delete(sm.Slots, k)
				}
			}
			return
		}
	}
}

func (sm *SessionManager) newSession(key string, claims *Claims) *Session {
	now := time.Now()
	session := &Session{
		Key:    key,
		Claims: claims,
	}
	session.InUse.Store(true)
	session.LastUsedAt = now
	list := sm.Slots[key]
	if list == nil {
		list = &SessionList{}
		sm.Slots[key] = list
	}
	list.append(session)
	return session
}

func (sm *SessionManager) getSession(key string, claims *Claims) (*Session, bool) {
	if !sm.enabled {
		return &Session{
			Key:    key,
			Claims: claims,
		}, true
	}
	sm.mtx.Lock()
	defer sm.mtx.Unlock()
	list := sm.Slots[key]
	if list == nil || list.isEmpty() {
		return sm.newSession(key, claims), true
	}
	session := list.Head
	swapped := session.InUse.CompareAndSwap(false, true)
	if !swapped {
		return sm.newSession(key, claims), true
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
	list := sm.Slots[session.Key]
	list.toFront(session)
	session.LastUsedAt = time.Now()
	swapped := session.InUse.CompareAndSwap(true, false)
	return swapped
}
