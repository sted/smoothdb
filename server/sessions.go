package server

import (
	"strconv"
	"time"
)

type Session struct {
	Id        string
	Role      string
	Token     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SessionManager struct {
	CurrentID uint64
	Sessions  map[string]*Session
}

func (s *SessionManager) NewSession(auth *Auth) *Session {
	s.CurrentID++
	now := time.Now()
	return &Session{
		Id:        strconv.FormatUint(s.CurrentID, 10),
		Role:      auth.Role,
		Token:     "",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (s *SessionManager) getSession(sessionId string) *Session {
	return s.Sessions[sessionId]
}
