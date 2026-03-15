package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

const logFileName = "databasus.log"

var (
	loggerInstance *slog.Logger
	once           sync.Once
)

func GetLogger() *slog.Logger {
	once.Do(func() {
		initialize()
	})

	return loggerInstance
}

func initialize() {
	writer := buildWriter()

	loggerInstance = slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format("2006/01/02 15:04:05"))
			}
			if a.Key == slog.LevelKey {
				return slog.Attr{}
			}

			return a
		},
	}))
}

func buildWriter() io.Writer {
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open %s for logging: %v\n", logFileName, err)
		return os.Stdout
	}

	return io.MultiWriter(os.Stdout, f)
}
