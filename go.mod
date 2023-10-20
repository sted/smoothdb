module github.com/smoothdb/smoothdb

go 1.21.0

require (
	github.com/golang-jwt/jwt/v4 v4.5.0
	github.com/imdario/mergo v0.3.16
	github.com/jackc/pgx/v5 v5.4.3
	github.com/rs/zerolog v1.31.0
	github.com/samber/lo v1.38.1
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	heligo v0.0.0-00010101000000-000000000000
)

replace heligo => ../../heligo

replace github.com/rs/zerolog v1.29.1 => github.com/sted/zerolog v0.0.0-20230413174247-61d5f1578065

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	golang.org/x/crypto v0.14.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/sync v0.4.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
)
