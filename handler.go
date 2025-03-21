package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func NewHandler(opts ...Option) (slog.Handler, error) {
	// Create a new config with default values
	cfg := DefaultConfig()

	// Apply options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate the configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	var handlers []slog.Handler

	// Create console handler
	if cfg.Console.Enabled {
		console, err := newConsoleHandler(cfg)
		if err != nil {
			return nil, fmt.Errorf("error creating console handler: %w", err)
		}
		handlers = append(handlers, console)
	}

	// Create file handler
	if cfg.File.Enabled && cfg.File.Path != "" {
		file, err := newFileHandler(cfg)
		if err != nil {
			return nil, fmt.Errorf("error creating file handler: %w", err)
		}
		handlers = append(handlers, file)
	}

	// No handlers, return a default console handler
	if len(handlers) == 0 {
		return slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     cfg.Level,
			AddSource: cfg.AddSource,
		}), nil
	}

	// Single handler, return it
	if len(handlers) == 1 {
		return handlers[0], nil
	}

	// Multiple handlers, return a multi handler
	return newMultiHandler(handlers...), nil
}

func newConsoleHandler(cfg *Config) (slog.Handler, error) {
	opts := &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	switch cfg.Console.Format {
	case FormatJSON:
		return slog.NewJSONHandler(os.Stderr, opts), nil
	case FormatText:
		return slog.NewTextHandler(os.Stderr, opts), nil
	case FormatCustom:
		return newCustomHandler(os.Stderr, cfg, &cfg.Console, opts)
	default:
		return nil, fmt.Errorf("unsupported log format: %v", cfg.Console.Format)
	}
}

func newFileHandler(cfg *Config) (slog.Handler, error) {
	writer, err := newRotatingWriter(&rotatingConfig{
		directory:     filepath.Dir(cfg.File.Path),
		fileName:      filepath.Base(cfg.File.Path),
		maxSizeMB:     cfg.File.MaxSizeMB,
		retentionDays: cfg.File.RetentionDays,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating rotating writer: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	switch cfg.File.Format {
	case FormatJSON:
		return slog.NewJSONHandler(writer, opts), nil
	case FormatText:
		return slog.NewTextHandler(writer, opts), nil
	case FormatCustom:
		return newCustomHandler(writer, cfg, &cfg.File, opts)
	default:
		return nil, fmt.Errorf("unsupported log format: %v", cfg.File.Format)
	}
}
