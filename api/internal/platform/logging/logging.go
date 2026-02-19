package logging

import (
	"log/slog"
	"os"
)

func Setup(level slog.Level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
