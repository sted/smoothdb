package jqeval

// Config holds the jq evaluation settings.
// It appears as the "JQ" section in the server configuration.
type Config struct {
	Enabled         bool `comment:"Enable jq evaluation: /jq route, jq= query parameter (default: false)"`
	Timeout         int  `comment:"Timeout in milliseconds for a single jq evaluation (default: 250)"`
	MaxProgramBytes int  `comment:"Maximum size in bytes for a jq program or its arguments (default: 4096)"`
	MaxUpdateRows   int  `comment:"Maximum number of rows updatable with a single jq update (default: 1000)"`
	CacheEntries    int  `comment:"Size of the compiled jq program cache (default: 256)"`
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:         false,
		Timeout:         250,
		MaxProgramBytes: 4096,
		MaxUpdateRows:   1000,
		CacheEntries:    256,
	}
}

var cfg = *DefaultConfig()

// Configure sets the package configuration and resets the compiled program
// cache. It is meant to be called once at startup, before serving requests.
// Zero or negative values fall back to the defaults (Enabled excepted).
func Configure(c *Config) {
	if c == nil {
		return
	}
	cfg = *c
	cache = newJQCache(cacheEntries())
}

// Enabled reports whether jq evaluation is enabled in the configuration
func Enabled() bool {
	return cfg.Enabled
}

// MaxUpdateRows returns the maximum number of rows a single jq update can affect
func MaxUpdateRows() int {
	if cfg.MaxUpdateRows > 0 {
		return cfg.MaxUpdateRows
	}
	return DefaultConfig().MaxUpdateRows
}

// MaxProgramBytes returns the maximum allowed size for a jq program.
// The same cap applies to the raw jq_args parameter.
func MaxProgramBytes() int {
	if cfg.MaxProgramBytes > 0 {
		return cfg.MaxProgramBytes
	}
	return DefaultConfig().MaxProgramBytes
}

func timeout() int {
	if cfg.Timeout > 0 {
		return cfg.Timeout
	}
	return DefaultConfig().Timeout
}

func cacheEntries() int {
	if cfg.CacheEntries > 0 {
		return cfg.CacheEntries
	}
	return DefaultConfig().CacheEntries
}
