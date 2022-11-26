package server

import (
	"strconv"
	"sync"
	"time"
)

type Session struct {
	Id         string
	Role       string
	Token      string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type SessionManager struct {
	CurrentID uint64
	Sessions  map[string]*Session
	mtx       sync.RWMutex
}

func (s *Server) initSessionManager() {
	s.sessionManager.Sessions = map[string]*Session{}
}

func (s *SessionManager) NewSession(auth *Auth) *Session {
	s.CurrentID++
	now := time.Now()
	session := &Session{
		Id:         strconv.FormatUint(s.CurrentID, 10),
		Role:       auth.Role,
		Token:      "",
		CreatedAt:  now,
		LastUsedAt: now,
	}
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.Sessions[session.Id] = session
	return session
}

func (s *SessionManager) getSession(sessionId string) *Session {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return s.Sessions[sessionId]
}
