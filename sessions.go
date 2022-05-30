package main

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
	CreateAt     time.Time
	LastUsedAt   time.Time
}

func NewSession() *Session {
	server.CurrentID = server.CurrentID + 1
	return &Session{
		Id:           fmt.Sprintf("%d", server.CurrentID),
		UserId:       "",
		UserName:     "",
		PasswordHash: "",
		Token:        "",
		CreateAt:     time.Now(),
		LastUsedAt:   time.Now(),
	}
}
