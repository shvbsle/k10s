package log

import (
	"io"
	"log/slog"
	"os"
)

type LoggerConfiguration struct {
	LogLevel slog.Level
	Writer   io.Writer
}

// SetupLogging configures logging for a given writer sink. It sets up a
// structured logger using slog.
func NewLogger(config *LoggerConfiguration) *slog.Logger {
	if config.Writer == nil {
		config.Writer = os.Stdout
	}

	return slog.New(slog.NewJSONHandler(config.Writer, &slog.HandlerOptions{
		Level:     config.LogLevel,
		AddSource: true,
	}))
}

// SetDefault sets the default logger
func SetDefault(logger *slog.Logger) {
	slog.SetDefault(logger)
}

// G returns the global logger instance
func G() *slog.Logger {
	return slog.Default()
}

// TUI returns a logger instance scoped with the TUI component
func TUI() *slog.Logger {
	return slog.With("component", "TUI")
}
