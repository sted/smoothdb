package server

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/smoothdb/smoothdb/config"
	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"

	"github.com/imdario/mergo"
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
	debug := os.Getenv("SMOOTHDB_DEBUG")
	if strings.ToLower(debug) == "true" {
		c.AllowAnon = true
		c.EnableAdminRoute = true
		c.Logging.Level = "trace"
		c.Logging.StdOut = true
	}
	enableAnon := os.Getenv("SMOOTHDB_ALLOW_ANON")
	if enableAnon != "" {
		if strings.ToLower(enableAnon) == "true" {
			c.AllowAnon = true
		} else {
			c.AllowAnon = false
		}
	}
	enableAdminRoute := os.Getenv("SMOOTHDB_ENABLE_ADMIN_ROUTE")
	if enableAdminRoute != "" {
		if strings.ToLower(enableAdminRoute) == "true" {
			c.EnableAdminRoute = true
		} else {
			c.EnableAdminRoute = false
		}
	}
}

const usageStr = `
Usage: smoothdb [options]

Server Options:
	-a, --addr <host>                Bind to host address (default: localhost:8081)
	-d, --dburl <url>                Database URL (default: postgres://localhost:5432)			
	-c, --config <file>              Configuration file (default: ./config.json)
	-h, --help                       Show this message
`

func getFlags(defaultConfigPath string) (*Config, string) {
	c := &Config{}
	var configPath string

	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("%s\n", usageStr)
		os.Exit(0)
	}
	flags.StringVar(&configPath, "c", defaultConfigPath, "Configuration file")
	flags.StringVar(&configPath, "config", defaultConfigPath, "Configuration file")
	flags.StringVar(&c.Address, "a", c.Address, "Address")
	flags.StringVar(&c.Address, "addr", c.Address, "Address")
	flags.StringVar(&c.Database.URL, "d", c.Database.URL, "DatabaseURL")
	flags.StringVar(&c.Database.URL, "dburl", c.Database.URL, "DatabaseURL")
	flags.Parse(os.Args[1:])
	return c, configPath
}

func getConfig(base *Config, opts *ConfigOptions) *Config {

	// Defaults
	cfg := defaultConfig()

	// Command line

	var defaultConfigPath string
	if opts == nil {
		defaultConfigPath = "./config.json"
	} else {
		defaultConfigPath = opts.ConfigFilePath
	}

	var cliConfig *Config
	var configPath string
	if opts == nil || !opts.SkipFlags {
		cliConfig, configPath = getFlags(defaultConfigPath)
	}
	// Configuration file
	config.GetConfig(cfg, configPath)

	if base != nil {
		mergo.Merge(cfg, base, mergo.WithOverride)
	}
	// Environment
	if opts == nil || !opts.SkipEnv {
		getEnvironment(cfg)
	}
	if opts == nil || !opts.SkipFlags {
		mergo.Merge(cfg, cliConfig, mergo.WithOverride)
	}

	return cfg
}
