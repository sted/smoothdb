package server

import (
	"encoding/json"
	"flag"
	"fmt"
	"green/green-ds/config"
	"green/green-ds/database"
	"green/green-ds/logging"
	"os"
	"strings"
)

// Config holds the current configuration
type Config struct {
	Address          string          `comment:"Server address and port (default localhost:8081)"`
	AllowAnon        bool            `comment:"Allow unauthenticated connections"`
	JWTSecret        string          `comment:"Secret for JWT tokens"`
	EnableAdminRoute bool            `comment:"Enable administration of databases and tables"`
	Database         database.Config `comment:"Database configuration"`
	Logging          logging.Config  `comment:"Logging configuration"`
}

func defaultConfig() *Config {
	return &Config{
		Address:          ":8081",
		AllowAnon:        false,
		JWTSecret:        "",
		EnableAdminRoute: false,
		Database:         *database.DefaultConfig(),
		Logging:          *logging.DefaultConfig(),
	}
}

func getEnvironment(c *Config) {
	dburl := os.Getenv("DATABASE_URL")
	if dburl != "" {
		c.Database.URL = dburl
	}
	debug := os.Getenv("GREEN_DEBUG")
	if strings.ToLower(debug) == "true" {
		c.AllowAnon = true
		c.EnableAdminRoute = true
		c.Logging.Level = "trace"
		c.Logging.StdOut = true
	}
	enableAnon := os.Getenv("GREEN_ALLOW_ANON")
	if enableAnon != "" {
		if strings.ToLower(enableAnon) == "true" {
			c.AllowAnon = true
		} else {
			c.AllowAnon = false
		}
	}
	enableAdminRoute := os.Getenv("GREEN_ENABLE_ADMIN_ROUTE")
	if enableAdminRoute != "" {
		if strings.ToLower(enableAdminRoute) == "true" {
			c.EnableAdminRoute = true
		} else {
			c.EnableAdminRoute = false
		}
	}
}

const usageStr = `
Usage: greenbase [options]

Server Options:
	-a, --addr <host>                Bind to host address (default: localhost:8081)
	-d, --dburl <url>                Database URL (default: postgres://localhost:5432)			
	-c, --config <file>              Configuration file (default: ./config.json)
	-h, --help                       Show this message
`

func getFlags(c *Config, configPath *string) *flag.FlagSet {
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("%s\n", usageStr)
		os.Exit(0)
	}
	flags.StringVar(configPath, "c", *configPath, "Configuration file")
	flags.StringVar(configPath, "config", *configPath, "Configuration file")
	flags.StringVar(&c.Address, "a", c.Address, "Address")
	flags.StringVar(&c.Address, "addr", c.Address, "Address")
	flags.StringVar(&c.Database.URL, "d", c.Database.URL, "DatabaseURL")
	flags.StringVar(&c.Database.URL, "dburl", c.Database.URL, "DatabaseURL")
	return flags
}

func getConfig(base *Config, configPath string) *Config {
	// Defaults
	c := defaultConfig()

	//
	if base != nil {
		b, _ := json.Marshal(base)
		_ = json.Unmarshal(b, c)
	}

	// Environment
	getEnvironment(c)

	// Command line flags
	flags := getFlags(c, &configPath)

	c = config.GetConfig(c, configPath)
	flags.Parse(os.Args[1:])

	return c
}
