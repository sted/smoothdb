package database

const DEFAULT_ANON = "anon"

type Config struct {
	URL                string   `comment:"Database URL"`
	MinPoolConnections int32    `comment:"Miminum connections per pool (default: 10)"`
	MaxPoolConnections int32    `comment:"Maximum connections per pool (default: 100)"`
	AnonRole           string   `comment:"Anonymous role (default: '' for no anon)"`
	AllowedDatabases   []string `comment:"Allowed databases (default: [] for all)"`
	SchemaSearchPath   []string `comment:"Schema search path (default: [] for Postgres search path)"`
	TransactionMode    string   `comment:"General transaction mode for operations: none, commit, commit-allow-override, rollback, rollback-allow-override (default: none)"`
	AggregatesEnabled  bool     `comment:"Enable aggregate functions (default: true)"`
}

func DefaultConfig() *Config {
	return &Config{
		URL:                "",
		MinPoolConnections: 10,
		MaxPoolConnections: 100,
		AnonRole:           "",
		TransactionMode:    "none",
		AggregatesEnabled:  true,
	}
}
