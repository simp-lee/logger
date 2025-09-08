package logger

import (
	"io"
	"log/slog"
)

// Logger wraps slog.Logger with automatic resource management
// By embedding *slog.Logger, it inherits all methods like Info, Error, Debug, Warn, With, WithGroup, etc.
type Logger struct {
	*slog.Logger
	closer io.Closer
}

// New creates a new Logger with automatic resource cleanup
// This is the recommended way to create a logger
func New(opts ...Option) (*Logger, error) {
	result, err := newHandler(opts...)
	if err != nil {
		return nil, err
	}
	return &Logger{
		Logger: slog.New(result.handler),
		closer: result.closer,
	}, nil
}

// Default returns a new Logger using the default slog configuration
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

// Close cleans up any resources held by the logger
// Always call this when you're done with the logger to prevent resource leaks
func (l *Logger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}
