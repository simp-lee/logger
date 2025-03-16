package logger

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// mockHandler is a mock implementation of slog.Handler for testing
type mockHandler struct {
	enabled bool
	handled bool
	attrs   []slog.Attr
	group   string
}

func (m *mockHandler) Enabled(context.Context, slog.Level) bool { return m.enabled }
func (m *mockHandler) Handle(context.Context, slog.Record) error {
	m.handled = true
	return nil
}
func (m *mockHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	m.attrs = attrs
	return m
}
func (m *mockHandler) WithGroup(name string) slog.Handler {
	m.group = name
	return m
}

func TestMultiHandler(t *testing.T) {
	t.Run("Enabled", func(t *testing.T) {
		h1 := &mockHandler{enabled: true}
		h2 := &mockHandler{enabled: false}
		mh := newMultiHandler(h1, h2)

		if !mh.Enabled(context.Background(), slog.LevelInfo) {
			t.Error("Expected multiHandler to be enabled")
		}

		h1.enabled = false
		if mh.Enabled(context.Background(), slog.LevelInfo) {
			t.Error("Expected multiHandler to be disabled")
		}
	})

	t.Run("Handle", func(t *testing.T) {
		h1 := &mockHandler{enabled: true}
		h2 := &mockHandler{enabled: true}
		mh := newMultiHandler(h1, h2)

		err := mh.Handle(context.Background(), slog.Record{})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !h1.handled || !h2.handled {
			t.Error("Expected both handlers to be called")
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		h1 := &mockHandler{}
		h2 := &mockHandler{}
		mh := newMultiHandler(h1, h2)

		attrs := []slog.Attr{slog.String("key", "value")}
		newMh := mh.WithAttrs(attrs)

		if len(h1.attrs) != 1 || len(h2.attrs) != 1 {
			t.Error("Expected attributes to be added to both handlers")
		}
		if _, ok := newMh.(*multiHandler); !ok {
			t.Error("Expected WithAttrs to return a multiHandler")
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		h1 := &mockHandler{}
		h2 := &mockHandler{}
		mh := newMultiHandler(h1, h2)

		newMh := mh.WithGroup("test_group")

		if h1.group != "test_group" || h2.group != "test_group" {
			t.Error("Expected group to be added to both handlers")
		}
		if _, ok := newMh.(*multiHandler); !ok {
			t.Error("Expected WithGroup to return a multiHandler")
		}
	})
}

func TestMultiError(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	me := &multiError{errors: []error{err1, err2}}

	expected := "error 1; error 2"
	if me.Error() != expected {
		t.Errorf("Expected error string %q, got %q", expected, me.Error())
	}
}
