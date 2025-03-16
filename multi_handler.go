package logger

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
)

type multiError struct {
	errors []error
}

func (e *multiError) Error() string {
	errStrings := make([]string, 0, len(e.errors))
	for _, err := range e.errors {
		if err != nil {
			errStrings = append(errStrings, err.Error())
		}
	}
	return strings.Join(errStrings, "; ")
}

// multiHandler is a custom slog.Handler that writes to multiple handlers
type multiHandler struct {
	handlers []slog.Handler
}

// newMultiHandler distributes records to multiple slog.Handler in parallel
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
	var wg sync.WaitGroup
	errs := make([]error, len(h.handlers))

	// Distribute the record to all handlers in parallel
	for i, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			wg.Add(1)
			go func(i int, h slog.Handler) {
				defer wg.Done()
				errs[i] = h.Handle(ctx, r.Clone())
			}(i, handler)
		}
	}
	wg.Wait()

	// Combine errors into a multiError
	filteredErrors := slices.DeleteFunc(errs, func(e error) bool {
		return e == nil
	})
	if len(filteredErrors) > 0 {
		return &multiError{errors: filteredErrors}
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
