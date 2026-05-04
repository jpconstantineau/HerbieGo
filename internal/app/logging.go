package app

import (
	"io"
	"log/slog"
	"os"
)

func loggerOrDiscard(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newProcessLogger() *slog.Logger {
	return newTextLogger(os.Stderr)
}

// NewDiscardLogger constructs a logger that drops process logs instead of
// writing into an active terminal UI.
func NewDiscardLogger() *slog.Logger {
	return newTextLogger(io.Discard)
}

func newTextLogger(writer io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}
