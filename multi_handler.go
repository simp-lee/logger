package logger

import (
	"context"
	"errors"
	"log/slog"
	"slices"
)

// multiHandler is a custom slog.Handler that writes to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler distributes records to multiple slog.Handler sequentially
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

// Enabled implements slog.Handler
func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle implements slog.Handler
func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error

	// Distribute the record to all handlers sequentially
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}

	// Combine errors into a multiError
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// WithAttrs implements slog.Handler
func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(slices.Clone(attrs))
	}
	return newMultiHandler(newHandlers...)
}

// WithGroup implements slog.Handler
func (h *multiHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return newMultiHandler(newHandlers...)
}
