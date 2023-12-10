package logging

type Config struct {
	Level         string `comment:"Log level: trace, debug, info, warn, error, fatal, panic (default info)"`
	FileLogging   bool   `comment:"Enable logging to file (default true)"`
	FilePath      string `comment:"File path for file-based logging (default smoothdb.log)"`
	MaxSize       int    `comment:"MaxSize is the maximum size in megabytes of the log file before it gets rotated."`
	MaxBackups    int    `comment:"MaxBackups is the maximum number of old log files to retain."`
	MaxAge        int    `comment:"MaxAge is the maximum number of days to retain old log files"`
	Compress      bool   `comment:"True to compress old log files (default false)"`
	StdOut        bool   `comment:"Enable logging to stdout (default false)"`
	PrettyConsole bool   `comment:"Enable pretty and colorful output for stdout (default false)"`
}

func DefaultConfig() *Config {
	return &Config{
		"info",
		true,
		"./smoothdb.log",
		25,
		2,
		5,
		false,
		false,
		false,
	}
}
