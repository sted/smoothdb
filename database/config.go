package database

type Config struct {
	URL                string   `comment:"Database URL (default: postgres://localhost:5432)"`
	MinPoolConnections int32    `comment:"Miminum connections per pool (default: 10)"`
	MaxPoolConnections int32    `comment:"Maximum connections per pool (default: 100)"`
	AnonRole           string   `comment:"Anonymous role (default: anon)"`
	AllowedDatabases   []string `comment:"Allowed databases (default: [] for all)"`
	SchemaSearchPath   []string `comment:"Schema search path (default: [] for Postgres search path)"`
	TransactionEnd     string   `comment:"How to end a tx: none, commit, rollback (default: none)"`
}

func DefaultConfig() *Config {
	return &Config{
		URL:                "postgres://localhost:5432/",
		MinPoolConnections: 10,
		MaxPoolConnections: 100,
		AnonRole:           "anon",
		TransactionEnd:     "none",
	}
}
