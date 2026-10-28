package database

import (
	"testing"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/rs/zerolog"
)

// pgx logs every query at its Info level, which DbLogger turns into a zerolog
// trace event. The tracer's threshold must follow that mapping: below a
// zerolog trace level it must not build the event data of every query only
// for zerolog to drop it.
func TestTracelogLevel(t *testing.T) {
	for _, c := range []struct {
		zl   zerolog.Level
		want tracelog.LogLevel
	}{
		{zerolog.TraceLevel, tracelog.LogLevelTrace},
		{zerolog.DebugLevel, tracelog.LogLevelWarn},
		{zerolog.InfoLevel, tracelog.LogLevelWarn},
		{zerolog.WarnLevel, tracelog.LogLevelWarn},
		{zerolog.ErrorLevel, tracelog.LogLevelError},
		{zerolog.FatalLevel, tracelog.LogLevelNone},
		{zerolog.Disabled, tracelog.LogLevelNone},
	} {
		if got := tracelogLevel(c.zl); got != c.want {
			t.Errorf("zerolog %v: tracelog level %v, want %v", c.zl, got, c.want)
		}
	}
}
