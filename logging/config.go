package logging

type Config struct {
	Level        string `comment:"Log level: trace, debug, info, warn, error, fatal, panic"`
	FilePath     string `comment:"File path for file-based logging. Use an empty string to disable logging to file"`
	MaxSize      int    `comment:"MaxSize is the maximum size in megabytes of the log file before it gets rotated."`
	MaxBackups   int    `comment:"MaxBackups is the maximum number of old log files to retain."`
	MaxAge       int    `comment:"MaxAge is the maximum number of days to retain old log files"`
	StdOut       bool   `comment:"Enable logging to stdout"`
	ConsoleColor bool   `comment:"Eanble pretty colorized output for stdout"`
}

func DefaultConfig() *Config {
	return &Config{
		"info",
		"./green-ds.log",
		25,
		2,
		5,
		false,
		false,
	}
}
