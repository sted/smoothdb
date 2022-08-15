package server

import (
	"fmt"
	"time"
)

type Session struct {
	Id           string
	UserId       string
	UserName     string
	PasswordHash string
	Token        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewSession() *Session {
	MainServer.CurrentID = MainServer.CurrentID + 1
	return &Session{
		Id:           fmt.Sprintf("%d", MainServer.CurrentID),
		UserId:       "",
		UserName:     "",
		PasswordHash: "",
		Token:        "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}
