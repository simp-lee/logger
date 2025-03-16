package logger

import (
	"log/slog"
)

// Logger is the interface for logging messages
// By embedding *slog.Logger,
// Logger automatically inherits all methods of slog.Logger
// including Info, Error, Debug, Warn, With, WithGroup, etc.,
// without needing to reimplement them
type Logger struct {
	*slog.Logger
}

// New creates a new Logger instance
// This Logger will delegate all logging operations to the provided slog.Handler
func New(handler slog.Handler) *Logger {
	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default returns a new Logger with the default slog logger
// This allows direct use of the standard library's default behavior
func Default() *Logger {
	return &Logger{
		Logger: slog.Default(),
	}
}

// SetDefault sets the current logger as the default logger
// After setting, standard library calls like slog.Info(), slog.Error() will use this logger
// Note: Since Logger embeds *slog.Logger, the following methods can be used directly without additional implementation:
// - With(args ...any): Adds attributes and returns a new Logger
// - WithGroup(name string): Adds a group and returns a new Logger
// - Info, Warn, Error, Debug and other log level methods
func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}
