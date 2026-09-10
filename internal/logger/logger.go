package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func New(level, format, output string) *slog.Logger {
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(getWriter(output), opts)
	} else {
		handler = slog.NewJSONHandler(getWriter(output), opts)
	}

	return slog.New(handler)
}

func getWriter(output string) io.Writer {
	switch strings.ToLower(output) {
	case "stderr":
		return os.Stderr
	case "file":
		f, err := os.OpenFile("atlas.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return os.Stdout
		}
		return f
	default:
		return os.Stdout
	}
}
