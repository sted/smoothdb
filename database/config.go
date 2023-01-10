package database

type Config struct {
	URL                string   `comment:"Database URL (default: postgres://localhost:5432)"`
	MinPoolConnections int32    `comment:"Miminum connections per pool (default: 10)"`
	MaxPoolConnections int32    `comment:"Maximum connections per pool (default: 100)"`
	AuthRole           string   `comment:"Authorization role (default: auth)"`
	AnonRole           string   `comment:"Anonymous role (default: anon)"`
	AllowedDatabases   []string `comment:"Allowed databases (default: [] for all)"`
	SchemaSearchPath   []string `comment:"Schema search path (default: [] for default Postgres search path)"`
}

func DefaultConfig() *Config {
	return &Config{
		URL:                "postgres://localhost:5432/",
		MinPoolConnections: 10,
		MaxPoolConnections: 100,
		AuthRole:           "auth",
		AnonRole:           "anon",
	}
}
