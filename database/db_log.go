// Adapted from https://github.com/jackc/pgx-zerolog

package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

type DbLogger struct {
	zlog zerolog.Logger
}

// NewDbLogger accepts a zerolog.Logger as input and returns a new custom pgx
// logging facade as output.
func NewDbLogger(logger *zerolog.Logger) *DbLogger {
	l := DbLogger{
		zlog: logger.With().Str("domain", "DB").Logger(),
	}
	return &l
}

func (pl *DbLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]interface{}) {
	var zlevel zerolog.Level
	switch level {
	case tracelog.LogLevelNone:
		zlevel = zerolog.NoLevel
	case tracelog.LogLevelError:
		zlevel = zerolog.ErrorLevel
	case tracelog.LogLevelWarn:
		zlevel = zerolog.WarnLevel
	case tracelog.LogLevelInfo:
		zlevel = zerolog.TraceLevel // info -> trace
	case tracelog.LogLevelDebug:
		zlevel = zerolog.DebugLevel
	case tracelog.LogLevelTrace:
		zlevel = zerolog.TraceLevel
	default:
		zlevel = zerolog.DebugLevel
	}

	event := pl.zlog.WithLevel(zlevel)
	if event.Enabled() {
		if data["time"] != nil {
			event.Dur("elapsed", data["time"].(time.Duration))
		}
		gctx := GetSmoothContext(ctx)
		var role string
		if gctx != nil {
			role = gctx.Role
		}
		event.Str("role", role)
		event.Str("method", "")
		event.Str("status", "")
		event.Fields(data)
		event.Msg(msg)
	}
}
