package logging

import (
	"testing"

	"github.com/rs/zerolog"
)

// InitLogger set only zerolog's global level and left the logger's own at
// trace, so GetLevel() always answered trace: whoever sized its work on the
// level (the pgx query tracer, the HTTP logger) did the trace-level work on
// every query and every request, only for the global level to drop the event.
func TestInitLoggerLevel(t *testing.T) {
	for _, c := range []struct {
		config string
		want   zerolog.Level
	}{
		{"info", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"trace", zerolog.TraceLevel},
		{"nonsense", zerolog.InfoLevel},
	} {
		logger := InitLogger(&Config{Level: c.config})
		if got := logger.GetLevel(); got != c.want {
			t.Errorf("Level %q: logger level %v, want %v", c.config, got, c.want)
		}
	}
}
