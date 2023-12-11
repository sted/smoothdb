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
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zerolog.Logger
}

// InitLogger initilaizes the logger
func InitLogger(config *Config) *Logger {
	var writers []io.Writer

	if config.FilePath != "" {
		writers = append(writers, newRollingFile(config))
	}
	if config.StdOut {
		if config.PrettyConsole {
			writers = append(writers, zerolog.ConsoleWriter{
				Out:           os.Stdout,
				TimeFormat:    time.DateTime,
				PartsOrder:    []string{"level", "time", "domain", "elapsed", "role", "method", "status", "message"},
				FormatPrepare: formatPrepare,
				FieldsExclude: []string{"domain", "elapsed", "role", "method", "status"},
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
	zlogger := zerolog.New(mw).With().Timestamp().Logger()

	return &Logger{&zlogger}
}

func formatPrepare(evt map[string]any) error {
	evt["domain"] = fmt.Sprintf("%-4s", evt["domain"])
	f, _ := strconv.ParseFloat(fmt.Sprint(evt["elapsed"]), 32)
	evt["elapsed"] = fmt.Sprintf("%8.3fms", f)
	evt["role"] = fmt.Sprintf("%-12s", evt["role"])
	s := evt["method"].(string)
	if len(s) != 0 {
		evt["method"] = fmt.Sprintf("%s%-7s%s", methodColor(s), s, reset)
	}
	s = fmt.Sprint(evt["status"]) // can be a number
	if len(s) != 0 {
		i, _ := strconv.ParseUint(s, 10, 32)
		evt["status"] = fmt.Sprintf("%s%3s%s", statusCodeColor(i), s, reset)
	}

	return nil
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

func statusCodeColor(code uint64) string {
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

func newRollingFile(config *Config) io.Writer {
	if err := os.MkdirAll(path.Dir(config.FilePath), 0744); err != nil {
		log.Error().Err(err).Str("path", path.Dir(config.FilePath)).Msg("can't create log directory")
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
