package server

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sted/smoothdb/config"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

type ConfigOptions struct {
	ConfigFilePath string
	SkipFlags      bool
	SkipEnv        bool
}

// Config holds the current configuration
type Config struct {
	Address          string          `comment:"Server address and port (default localhost:8082)"`
	AllowAnon        bool            `comment:"Allow unauthenticated connections"`
	JWTSecret        string          `comment:"Secret for JWT tokens"`
	EnableAdminRoute bool            `comment:"Enable administration of databases and tables (default: false)"`
	SessionMode      string          `comment:"Session mode: none, role (default: role)"`
	BaseAPIURL       string          `comment:"Base URL for the API (default: /api)"`
	ShortAPIURL      bool            `comment:"Avoid database name in API URL (needs a single allowed database)"`
	BaseAdminURL     string          `comment:"Base URL for the Admin API (default: /admin)"`
	Database         database.Config `comment:"Database configuration"`
	Logging          logging.Config  `comment:"Logging configuration"`
}

func defaultConfig() *Config {
	return &Config{
		Address:          ":8081",
		AllowAnon:        false,
		JWTSecret:        "",
		EnableAdminRoute: false,
		SessionMode:      "role",
		BaseAPIURL:       "/api",
		ShortAPIURL:      false,
		BaseAdminURL:     "/admin",
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
		c.AllowAnon = strings.ToLower(enableAnon) == "true"
	}
	enableAdminRoute := os.Getenv("SMOOTHDB_ENABLE_ADMIN_ROUTE")
	if enableAdminRoute != "" {
		c.EnableAdminRoute = strings.ToLower(enableAdminRoute) == "true"
	}
}

const usageStr = `
Usage: smoothdb [options]

Server Options:
	-a, --addr <host>                Bind to host address (default: localhost:8081)
	-d, --dburl <url>                Database URL (default: postgres://localhost:5432)			
	-c, --config <file>              Configuration file (default: ./config.jsonc)
	-h, --help                       Show this message
`

func getFlags(defaultConfigPath string) (map[string]any, string) {
	var configPath, address, dburl string
	flags := flag.NewFlagSet("", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("%s\n", usageStr)
		os.Exit(0)
	}
	flags.StringVar(&configPath, "c", defaultConfigPath, "Configuration file")
	flags.StringVar(&configPath, "config", defaultConfigPath, "Configuration file")
	flags.StringVar(&address, "a", "", "Address")
	flags.StringVar(&address, "addr", "", "Address")
	flags.StringVar(&dburl, "d", "", "DatabaseURL")
	flags.StringVar(&dburl, "dburl", "", "DatabaseURL")
	flags.Parse(os.Args[1:])
	m := map[string]any{}
	return m, configPath
}

func getConfig(baseConfig map[string]any, configOpts *ConfigOptions) *Config {
	// Defaults
	cfg := defaultConfig()

	var defaultConfigPath string
	if configOpts == nil || configOpts.ConfigFilePath == "" {
		defaultConfigPath = "./config.jsonc"
	} else {
		defaultConfigPath = configOpts.ConfigFilePath
	}
	var cliConfig map[string]any
	configPath := defaultConfigPath
	if configOpts == nil || !configOpts.SkipFlags {
		cliConfig, configPath = getFlags(defaultConfigPath)
	}
	// Configuration file
	config.GetConfig(cfg, configPath)
	// Merge base config
	if baseConfig != nil {
		config.MergeConfig(cfg, baseConfig)
	}
	// Environment
	if configOpts == nil || !configOpts.SkipEnv {
		getEnvironment(cfg)
	}
	// Command line
	if configOpts == nil || !configOpts.SkipFlags {
		config.MergeConfig(cfg, cliConfig)
	}

	return cfg
}

func checkConfig(cfg *Config) error {
	if cfg.ShortAPIURL && len(cfg.Database.AllowedDatabases) != 1 {
		fmt.Println("Warning: Cannot enable ShortAPIURL with Database.AllowedDatabases is not configured with a single db")
		cfg.ShortAPIURL = false
	}
	return nil
}
