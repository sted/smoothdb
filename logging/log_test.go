package logging

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/rs/zerolog"
)

// newRollingFile returned nil when the log directory could not be created,
// and InitLogger appended the nil writer: the first log line panicked inside
// the MultiLevelWriter.
func TestUnwritableLogDirectoryDoesNotPanic(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	// a directory under a regular file cannot be created
	logger := InitLogger(&Config{Level: "info", FileLogging: true, FilePath: filepath.Join(blocker, "sub", "smoothdb.log")})
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("logging panicked: %v", r)
		}
	}()
	logger.Info().Msg("first line")
}

// The log directory was created with mode 0744: no one but the owner could
// enter it, so a log shipper running as another user could not read the log.
func TestLogDirectoryMode(t *testing.T) {
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	dir := filepath.Join(t.TempDir(), "logs")
	logger := InitLogger(&Config{Level: "info", FileLogging: true, FilePath: filepath.Join(dir, "smoothdb.log")})
	logger.Info().Msg("first line")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0755)&^os.FileMode(umask); got != want {
		t.Errorf("log directory mode %v, want %v", got, want)
	}
}

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
