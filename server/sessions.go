package server

import (
	"context"
	"green/green-ds/database"
	"green/green-ds/logging"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Session struct {
	Id         string
	Token      string
	Role       string
	InUse      atomic.Bool
	CreatedAt  time.Time
	LastUsedAt time.Time
	DbConn     *database.DbConn
}

type SessionManager struct {
	CurrentID uint64
	Sessions  map[string]*Session
	mtx       sync.RWMutex
	logger    *logging.Logger
}

func (s *Server) initSessionManager() {
	sm := &s.sessionManager
	sm.Sessions = map[string]*Session{}
	sm.logger = s.Logger

	go func() {
		ticker := time.NewTicker(1 * time.Second)

		for {
			select {
			case <-ticker.C:
				now := time.Now()
				sm.mtx.Lock()

				for k, s := range sm.Sessions {
					if s.InUse.Load() {
						continue
					}
					// Here we have a session not in use, which cannot be used now
					// because we hold a W lock on the session manager and making
					// the session usable requires an R lock

					passedTime := now.Sub(s.LastUsedAt)

					if passedTime > 5*time.Second {

						// Delete the session
						delete(sm.Sessions, k)

					} else if passedTime > 1*time.Second && s.DbConn != nil {

						// Release and detach the database connection from the session
						// (Acquire and attach are done in the auth middleware)
						err := database.ReleaseConnection(context.Background(), s.DbConn, true)
						sm.logger.Err(err).Msg("error releasing an expired session")
						s.DbConn = nil
					}
				}

				sm.mtx.Unlock()

			case <-s.shutdown:
				for k, s := range sm.Sessions {
					if s.DbConn != nil {
						database.ReleaseConnection(context.Background(), s.DbConn, false)
					}
					delete(sm.Sessions, k)
				}
				return
			}
		}
	}()
}

func (s *SessionManager) newSession(auth *Auth) *Session {
	now := time.Now()
	session := &Session{
		Role:      auth.Role,
		CreatedAt: now,
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.CurrentID++
	session.Id = strconv.FormatUint(s.CurrentID, 10)
	session.InUse.Store(true)
	session.LastUsedAt = now
	s.Sessions[session.Id] = session
	return session
}

func (s *SessionManager) getSession(sessionId string) *Session {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	session := s.Sessions[sessionId]
	if session == nil {
		return nil
	}
	swapped := session.InUse.CompareAndSwap(false, true)
	if !swapped {
		return nil
	}
	return session
}

func (s *SessionManager) leaveSession(session *Session) bool {
	now := time.Now()
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	swapped := session.InUse.CompareAndSwap(true, false)
	if !swapped {
		return false
	}
	session.LastUsedAt = now
	return true
}
