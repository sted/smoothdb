package server

import (
	"green/green-ds/database"
	"green/green-ds/logging"
)

// Config holds the current configuration
type Config struct {
	Address     string          `comment:"Server address and port (default localhost:8081)"`
	AllowAnon   bool            `comment:"Allow unauthenticated connections"`
	JWTSecret   string          `comment:"Secret for JWT tokens"`
	EnableAdmin bool            `comment:"Enable administration of databases and tables"`
	Database    database.Config `comment:"Database configuration"`
	Logging     logging.Config  `comment:"Logging configuration"`
}

func DefaultConfig() *Config {
	return &Config{
		Address:     ":8081",
		AllowAnon:   false,
		JWTSecret:   "",
		EnableAdmin: false,
		Database:    *database.DefaultConfig(),
		Logging:     *logging.DefaultConfig(),
	}
}
