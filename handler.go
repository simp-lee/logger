package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

// handlerResult holds a handler and its associated closer
type handlerResult struct {
	handler slog.Handler
	closer  io.Closer
}

// newHandler creates a handler with resource management
func newHandler(opts ...Option) (*handlerResult, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	var handlers []slog.Handler
	var closers []io.Closer

	// Console handler
	if cfg.Console.Enabled {
		handler, err := newConsoleHandler(cfg)
		if err != nil {
			return nil, fmt.Errorf("console handler error: %w", err)
		}
		handlers = append(handlers, handler)
	}

	// File handler
	if cfg.File.Enabled && cfg.File.Path != "" {
		handler, closer, err := newFileHandler(cfg)
		if err != nil {
			return nil, fmt.Errorf("file handler error: %w", err)
		}
		handlers = append(handlers, handler)
		if closer != nil {
			closers = append(closers, closer)
		}
	}

	// Default to console if no handlers
	if len(handlers) == 0 {
		return &handlerResult{
			handler: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level:     cfg.Level,
				AddSource: cfg.AddSource,
			}),
		}, nil
	}

	var combinedCloser io.Closer
	if len(closers) > 0 {
		combinedCloser = &multiCloser{closers: closers}
	}

	// Single handler
	if len(handlers) == 1 {
		return &handlerResult{
			handler: handlers[0],
			closer:  combinedCloser,
		}, nil
	}

	// Multiple handlers
	return &handlerResult{
		handler: newMultiHandler(handlers...),
		closer:  combinedCloser,
	}, nil
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
		return nil, fmt.Errorf("unsupported console format: %v", cfg.Console.Format)
	}
}

func newFileHandler(cfg *Config) (slog.Handler, io.Closer, error) {
	writer, err := newRotatingWriter(&rotatingConfig{
		directory:     filepath.Dir(cfg.File.Path),
		fileName:      filepath.Base(cfg.File.Path),
		maxSizeMB:     cfg.File.MaxSizeMB,
		retentionDays: cfg.File.RetentionDays,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("rotating writer error: %w", err)
	}

	opts := &slog.HandlerOptions{
		Level:       cfg.Level,
		AddSource:   cfg.AddSource,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	var handler slog.Handler
	switch cfg.File.Format {
	case FormatJSON:
		handler = slog.NewJSONHandler(writer, opts)
	case FormatText:
		handler = slog.NewTextHandler(writer, opts)
	case FormatCustom:
		h, err := newCustomHandler(writer, cfg, &cfg.File, opts)
		if err != nil {
			writer.Close()
			return nil, nil, err
		}
		handler = h
	default:
		writer.Close()
		return nil, nil, fmt.Errorf("unsupported file format: %v", cfg.File.Format)
	}

	return handler, writer, nil
}

// multiCloser closes multiple closers
type multiCloser struct {
	closers []io.Closer
}

func (mc *multiCloser) Close() error {
	var firstErr error
	for _, closer := range mc.closers {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
