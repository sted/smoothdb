package logging

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zerolog.Logger
}

// InitLogger initilaizes the logger
func InitLogger(config *Config) *Logger {
	var writers []io.Writer

	if config.FileLogging && config.FilePath != "" {
		if file := newRollingFile(config); file != nil {
			writers = append(writers, file)
		}
	}
	if config.StdOut {
		if config.PrettyConsole {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:           os.Stdout,
				TimeFormat:    time.DateTime,
				PartsOrder:    []string{"level", "time", "domain", "elapsed", "role", "method", "status", "message"},
				FormatPrepare: createFormatPrepare(config.ColorConsole),
				FieldsExclude: []string{"domain", "elapsed", "role", "method", "status"},
				NoColor:       !config.ColorConsole,
			})
		} else {
			writers = append(writers, os.Stdout)
		}
	}
	mw := zerolog.MultiLevelWriter(writers...)

	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	// the logger's own level too: GetLevel() answers it, and the query tracer
	// and the HTTP logger size their work on it
	zlogger := zerolog.New(mw).Level(level).With().Timestamp().Logger()

	return &Logger{&zlogger}
}

func createFormatPrepare(withColor bool) func(map[string]any) error {
	return func(evt map[string]any) error {
		domain, _ := evt["domain"].(string)
		evt["domain"] = fmt.Sprintf("%-4s", domain)

		v := evt["elapsed"]
		if v != nil {
			elapsed := fmt.Sprint(v)
			if f, err := strconv.ParseFloat(elapsed, 32); err == nil {
				evt["elapsed"] = fmt.Sprintf("%8.3fms", f)
			} else {
				evt["elapsed"] = fmt.Sprintf("%10s", "")
			}
		} else {
			evt["elapsed"] = fmt.Sprintf("%10s", "")
		}
		role, _ := evt["role"].(string)
		evt["role"] = fmt.Sprintf("%-12s", role)

		if method, ok := evt["method"].(string); ok && withColor {
			evt["method"] = fmt.Sprintf("%s%-7s%s", methodColor(method), method, reset)
		} else {
			evt["method"] = fmt.Sprintf("%-7s", method)
		}

		v = evt["status"]
		if v != nil {
			status := fmt.Sprint(v)
			if i, err := strconv.ParseInt(status, 10, 32); err == nil && withColor {
				evt["status"] = fmt.Sprintf("%s%3s%s", statusCodeColor(i), status, reset)
			} else {
				evt["status"] = fmt.Sprintf("%3s", status)
			}
		} else {
			evt["status"] = fmt.Sprintf("%3s", "")
		}

		return nil
	}
}

const (
	green   = "\033[97;42m"
	white   = "\033[97;47m"
	yellow  = "\033[97;43m"
	red     = "\033[97;41m"
	blue    = "\033[97;44m"
	magenta = "\033[97;45m"
	cyan    = "\033[97;46m"
	reset   = "\033[0m"
)

func statusCodeColor(code int64) string {
	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return green
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return white
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return yellow
	default:
		return red
	}
}

func methodColor(method string) string {
	switch method {
	case http.MethodGet:
		return blue
	case http.MethodPost:
		return cyan
	case http.MethodPut:
		return yellow
	case http.MethodDelete:
		return red
	case http.MethodPatch:
		return green
	case http.MethodHead:
		return magenta
	case http.MethodOptions:
		return white
	default:
		return reset
	}
}

// newRollingFile returns the log file writer, or nil when its directory
// cannot be created: file logging is then skipped, with the error on stderr
// (the logger being built cannot report it).
func newRollingFile(config *Config) io.Writer {
	dir := path.Dir(config.FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot create the log directory %s, file logging disabled: %v\n", dir, err)
		return nil
	}
	return &lumberjack.Logger{
		Filename:   path.Join(config.FilePath),
		MaxBackups: config.MaxBackups,
		MaxSize:    config.MaxSize,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}
}
