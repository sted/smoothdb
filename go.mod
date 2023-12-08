module github.com/sted/smoothdb

go 1.21.0

require (
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/jackc/pgx/v5 v5.5.0
	github.com/rs/zerolog v1.31.0
	github.com/samber/lo v1.39.0
	github.com/sted/heligo v0.1.1
	github.com/tailscale/hujson v0.0.0-20221223112325-20486734a56a
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

replace github.com/rs/zerolog v1.31.0 => github.com/sted/zerolog v0.0.0-20230413174247-61d5f1578065

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	golang.org/x/crypto v0.16.0 // indirect
	golang.org/x/exp v0.0.0-20231206192017-f3f8817b8deb // indirect
	golang.org/x/sync v0.5.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)
