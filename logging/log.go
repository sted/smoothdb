package logging

import (
	"io"
	"os"
	"path"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	*zerolog.Logger
}

// InitLogger initilaizes the wsbus logger
func InitLogger(config *Config) *Logger {
	var writers []io.Writer

	if config.FilePath != "" {
		writers = append(writers, newRollingFile(config))
	}
	if config.StdOut {
		if config.ConsoleColor {
			writers = append(writers, zerolog.ConsoleWriter{Out: os.Stdout})
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
	}
}
