package server

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
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
	Address              string          `comment:"Server address and port (default: 0.0.0.0:4000)"`
	CertFile             string          `comment:"TLS certificate file (default: '')"`
	KeyFile              string          `comment:"TLS certificate key file (default: '')"`
	AllowAnon            bool            `comment:"Allow unauthenticated connections (default: false)"`
	JWTSecret            string          `comment:"Secret for JWT tokens"`
	SessionMode          string          `comment:"Session mode: none, role (default: role)"`
	EnableAdminRoute     bool            `comment:"Enable administration of databases and tables (default: false)"`
	EnableAdminUI        bool            `comment:"Enable Admin dashboard (default: false)"`
	EnableAPIRoute       bool            `comment:"Enable API access (default: true)"`
	BaseAPIURL           string          `comment:"Base URL for the API (default: /api)"`
	ShortAPIURL          bool            `comment:"Avoid database name in API URL (needs a single allowed database)"`
	BaseAdminURL         string          `comment:"Base URL for the Admin API (default: /admin)"`
	CORSAllowedOrigins   []string        `comment:"CORS Access-Control-Allow-Origin (default: [*] for all)"`
	CORSAllowCredentials bool            `comment:"CORS Access-Control-Allow-Credentials (default: false)"`
	EnableDebugRoute     bool            `comment:"Enable debug access (default: false)"`
	PluginDir            string          `comment:"Plugins' directory (default: ./_plugins)"`
	Plugins              []string        `comment:"Ordered list of plugins (default: [])"`
	ReadTimeout          int64           `comment:"The maximum duration (seconds) for reading the entire request, including the body (default: 60)"`
	WriteTimeout         int64           `comment:"The maximum duration before timing out writes of the response (default: 60)"`
	RequestMaxBytes      int64           `comment:"Max bytes allowed in requests, to limit the size of incoming request bodies (default: 1M, 0 for unlimited)"`
	Database             database.Config `comment:"Database configuration"`
	Logging              logging.Config  `comment:"Logging configuration"`
}

func defaultConfig() *Config {
	return &Config{
		Address:              "0.0.0.0:4000",
		CertFile:             "",
		KeyFile:              "",
		AllowAnon:            false,
		JWTSecret:            "",
		SessionMode:          "role",
		EnableAdminRoute:     false,
		EnableAdminUI:        false,
		EnableAPIRoute:       true,
		BaseAPIURL:           "/api",
		ShortAPIURL:          false,
		BaseAdminURL:         "/admin",
		CORSAllowedOrigins:   []string{"*"},
		CORSAllowCredentials: false,
		EnableDebugRoute:     false,
		PluginDir:            "./_plugins",
		Plugins:              []string{},
		ReadTimeout:          60,
		WriteTimeout:         60,
		RequestMaxBytes:      1024 * 1024,
		Database:             *database.DefaultConfig(),
		Logging:              *logging.DefaultConfig(),
	}
}

func getEnvironment(c *Config) {
	dburl := os.Getenv("SMOOTHDB_DATABASE_URL")
	if dburl != "" {
		c.Database.URL = dburl
	}
	jwtSecret := os.Getenv("SMOOTHDB_JWT_SECRET")
	if jwtSecret != "" {
		c.JWTSecret = jwtSecret
	}
	debug := os.Getenv("SMOOTHDB_DEBUG")
	if strings.ToLower(debug) == "true" {
		c.AllowAnon = true
		c.EnableAdminRoute = true
		c.EnableAdminUI = true
		c.Logging.Level = "trace"
		c.Logging.StdOut = true
		c.EnableDebugRoute = true
	}
	allowAnon := os.Getenv("SMOOTHDB_ALLOW_ANON")
	if allowAnon != "" {
		c.AllowAnon = strings.ToLower(allowAnon) == "true"
	}
	enableAdminRoute := os.Getenv("SMOOTHDB_ENABLE_ADMIN_ROUTE")
	if enableAdminRoute != "" {
		c.EnableAdminRoute = strings.ToLower(enableAdminRoute) == "true"
	}
}

const usageStr = `
Usage: smoothdb [options]

Server Options:
	-a, --addr <host>    Bind to host address (default: '0.0.0.0:4000')
	-d, --dburl <dburl>  Database URL			
	-c, --config <path>  Configuration file (default: './config.jsonc')
	--initdb             Initialize db interactively and exit
	-h, --help           Show this message
`

func getFlags(defaultConfigPath string) (map[string]any, string, bool) {
	var configPath, address, dburl string
	var initdb bool
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("%s\n", usageStr)
		os.Exit(0)
	}
	flags.StringVar(&configPath, "c", defaultConfigPath, "")
	flags.StringVar(&configPath, "config", defaultConfigPath, "")
	flags.StringVar(&address, "a", "", "")
	flags.StringVar(&address, "addr", "", "")
	flags.StringVar(&dburl, "d", "", "")
	flags.StringVar(&dburl, "dburl", "", "")
	flags.BoolVar(&initdb, "initdb", false, "")
	flags.Parse(os.Args[1:])
	m := map[string]any{}
	if address != "" {
		m["Address"] = address
	}
	if dburl != "" {
		m["Database.URL"] = dburl
	}
	return m, configPath, initdb
}

func getConfig(baseConfig map[string]any, configOpts *ConfigOptions) (*Config, error) {
	// Defaults
	defaultCfg := defaultConfig()

	var defaultConfigPath string
	if configOpts == nil || configOpts.ConfigFilePath == "" {
		defaultConfigPath = "./config.jsonc"
	} else {
		defaultConfigPath = configOpts.ConfigFilePath
	}
	var cliConfig map[string]any
	configPath := defaultConfigPath
	var initdb bool
	if configOpts == nil || !configOpts.SkipFlags {
		cliConfig, configPath, initdb = getFlags(defaultConfigPath)
	}
	// Configuration file
	cfg, err := config.GetConfig(defaultCfg, configPath)
	if err != nil {
		return nil, err
	}
	// Merge base config
	if baseConfig != nil {
		err = config.MergeConfig(cfg, baseConfig)
		if err != nil {
			return nil, err
		}
	}
	// Environment
	if configOpts == nil || !configOpts.SkipEnv {
		getEnvironment(cfg)
	}
	// Merge command line config
	if configOpts == nil || !configOpts.SkipFlags {
		err = config.MergeConfig(cfg, cliConfig)
		if err != nil {
			return nil, err
		}
	}
	if initdb {
		adminURL, dbcfg, err := askDbConfig()
		err = config.MergeConfig(cfg, dbcfg)
		if err != nil {
			return nil, err
		}
		err = database.PrepareDatabase(adminURL, &cfg.Database)
		if err != nil {
			return nil, err
		}
		config.SaveConfig(cfg, configPath)
		os.Exit(0)
	}
	return cfg, nil
}

func checkConfig(cfg *Config) error {
	if cfg.ShortAPIURL && len(cfg.Database.AllowedDatabases) != 1 {
		fmt.Println("Warning: 'ShortAPIURL' requires a single db in 'Database.AllowedDatabases'")
		cfg.ShortAPIURL = false
	}
	canContinue, err := database.CheckDatabase(&cfg.Database)
	if err != nil {
		return err
	}
	if !canContinue {
		return fmt.Errorf("Exiting")
	}
	return nil
}

func askOption(description, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	if defaultValue != "" {
		description += " (press enter for the default '%s') "
		fmt.Printf(description, defaultValue)
	} else {
		fmt.Print(description)
	}
	value, _ := reader.ReadString('\n')
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	return value
}

func askDbConfig() (string, map[string]any, error) {
	const (
		defaultAdminURL = "postgresql://postgres:postgres@localhost:5432"
		defaultAuth     = "auth"
	)
	var (
		adminURL string
		pgconfig *database.PostgresConfig
		err      error
	)
	config := map[string]any{}

	for {
		adminURL = askOption("Database URL for administration: ", defaultAdminURL)
		pgconfig, err = database.ParsePostgresURL(adminURL)
		if err == nil {
			break
		} else {

		}
	}

	authUser := askOption("Authenticator user: ", defaultAuth)
	authPassword := askOption("Authenticator password: ", "")
	config["Database.URL"] = "postgresql://" + authUser + ":" + authPassword + "@" +
		pgconfig.Host + ":" + strconv.Itoa(int(pgconfig.Port)) + "/" + database.SMOOTHDB
	config["Database.AnonRole"] = askOption("Anonymous user: ", database.DEFAULT_ANON)

	return adminURL, config, nil
}
