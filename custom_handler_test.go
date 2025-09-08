package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// mockOutputConfig implements outputConfig interface for testing
type mockOutputConfig struct {
	format    OutputFormat
	color     bool
	formatter string
}

func (m *mockOutputConfig) GetFormat() OutputFormat {
	return m.format
}

func (m *mockOutputConfig) GetColor() bool {
	return m.color
}

func (m *mockOutputConfig) GetFormatter() string {
	return m.formatter
}

func TestCustomHandler(t *testing.T) {
	t.Run("BasicFormatting", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false, // Disable color for easier testing
			formatter: "{level} {message} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		record := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "test message",
		}
		record.AddAttrs(slog.String("key", "value"))

		err = handler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "INFO") {
			t.Errorf("Output doesn't contain level: %q", output)
		}
		if !strings.Contains(output, "test message") {
			t.Errorf("Output doesn't contain message: %q", output)
		}
		if !strings.Contains(output, "key=value") {
			t.Errorf("Output doesn't contain attribute: %q", output)
		}
	})

	t.Run("WithAttrs", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{message} {attrs}",
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, nil)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		// Add attributes to handler
		newHandler := handler.WithAttrs([]slog.Attr{slog.String("handler_attr", "handler_val")})

		record := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "test message",
		}
		record.AddAttrs(slog.String("record_attr", "record_val"))

		err = newHandler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "handler_attr=handler_val") {
			t.Errorf("Output doesn't contain handler attribute: %q", output)
		}
		if !strings.Contains(output, "record_attr=record_val") {
			t.Errorf("Output doesn't contain record attribute: %q", output)
		}
	})

	t.Run("WithGroup", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{message} {attrs}",
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, nil)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		// Add group to handler
		newHandler := handler.WithGroup("test_group")

		record := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "test message",
		}

		err = newHandler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "test_group: test message") {
			t.Errorf("Output doesn't contain group prefix: %q", output)
		}
	})

	t.Run("ColorFormatting", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     true, // Enable color for testing
			formatter: "{level} {message}",
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, nil)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		record := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelError,
			Message: "error message",
		}

		err = handler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		// Check if ANSI color codes are present
		if !strings.Contains(output, "\033[") {
			t.Errorf("Color formatting not applied: %q", output)
		}
	})

	t.Run("FileOutputNoColor", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		// Use real FileConfig for testing
		outputCfg := &cfg.File
		outputCfg.Format = FormatCustom
		outputCfg.Formatter = "{level} {message}"

		handler, err := newCustomHandler(&buf, cfg, outputCfg, nil)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		record := slog.Record{
			Time:    time.Now(),
			Level:   slog.LevelError,
			Message: "error message",
		}

		err = handler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		// Check if ANSI color codes are not present
		if strings.Contains(output, "\033[") {
			t.Errorf("File output should not have color codes: %q", output)
		}
	})
}

func TestCustomHandler_ReplaceAttr(t *testing.T) {
	var buf bytes.Buffer

	// Create a config with ReplaceAttr function
	cfg := DefaultConfig()
	cfg.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// Replace sensitive data
		if a.Key == "password" {
			return slog.String("password", "***")
		}
		// Transform specific keys
		if a.Key == "user_id" {
			return slog.String("uid", a.Value.String())
		}
		return a
	}

	outputCfg := &mockOutputConfig{
		format:    FormatCustom,
		color:     false,
		formatter: "{message} {attrs}",
	}

	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	record := slog.Record{
		Level:   slog.LevelInfo,
		Message: "test login",
	}
	record.AddAttrs(
		slog.String("password", "secret123"),
		slog.String("user_id", "12345"),
		slog.String("username", "john"),
	)

	err = handler.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	output := buf.String()
	t.Logf("Output: %q", output)

	// Check if ReplaceAttr was applied
	if strings.Contains(output, "secret123") {
		t.Error("ReplaceAttr not applied: password should be masked")
	}
	if !strings.Contains(output, "password=***") {
		t.Error("ReplaceAttr not applied: password should be '***'")
	}
	if !strings.Contains(output, "uid=12345") {
		t.Error("ReplaceAttr not applied: user_id should be transformed to uid")
	}
	if strings.Contains(output, "user_id") {
		t.Error("ReplaceAttr not applied: user_id key should be replaced")
	}
}

func TestSpacePreservation(t *testing.T) {
	t.Run("MessageWithMultipleSpaces", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{message} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		logger := slog.New(handler)

		// Test message with multiple spaces
		testMessage := "this   has    multiple     spaces"
		logger.Info(testMessage)

		output := buf.String()
		t.Logf("Output: %q", output)

		// Check if multiple spaces are preserved
		if !strings.Contains(output, "   ") {
			t.Error("Multiple spaces in message were not preserved")
		}

		// The original message should be intact
		if !strings.Contains(output, testMessage) {
			t.Errorf("Original message %q not found in output %q", testMessage, output)
		}
	})

	t.Run("AttributeValueWithMultipleSpaces", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{message} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		logger := slog.New(handler)

		// Test attribute value with multiple spaces
		attrValue := "value   with    multiple     spaces"
		logger.Info("test message", "key", attrValue)

		output := buf.String()
		t.Logf("Output: %q", output)

		// Check if multiple spaces in attribute value are preserved
		if !strings.Contains(output, "   ") {
			t.Error("Multiple spaces in attribute value were not preserved")
		}

		// The original attribute value should be intact
		if !strings.Contains(output, attrValue) {
			t.Errorf("Original attribute value %q not found in output %q", attrValue, output)
		}
	})

	t.Run("JsonStringWithSpaces", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{message} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		logger := slog.New(handler)

		// Test with JSON-like string that has important spaces
		jsonStr := `{"name": "John   Doe", "address": "123   Main    St"}`
		logger.Info("processing json", "data", jsonStr)

		output := buf.String()
		t.Logf("Output: %q", output)

		// The JSON string should preserve its internal spaces
		if !strings.Contains(output, jsonStr) {
			t.Errorf("Original JSON string %q not found in output %q", jsonStr, output)
		}
	})
}

func TestEmptyPlaceholderHandling(t *testing.T) {
	t.Run("EmptyFileAndAttrs", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format: FormatCustom,
			color:  false,
			// This format has spaces around placeholders that could become empty
			formatter: "{level} {message} {file} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false, // This makes {file} empty
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		logger := slog.New(handler)

		// Log message without attributes - both {file} and {attrs} should be empty
		logger.Info("test message")

		output := buf.String()
		t.Logf("Output: %q", output)

		// Should not have excessive spaces
		if strings.Contains(output, "  ") {
			t.Error("Found double spaces in output, empty placeholder cleanup may not be working")
		}

		// Should contain the message
		if !strings.Contains(output, "test message") {
			t.Error("Output should contain the message")
		}

		// Should be properly formatted (level + message)
		expected := "INFO test message"
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, output)
		}
	})

	t.Run("OnlyAttrsEmpty", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{level} {message} {attrs}",
		}

		opts := &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
		}

		handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		logger := slog.New(handler)

		// Log message without attributes
		logger.Info("test message")

		output := buf.String()
		t.Logf("Output: %q", output)

		// Should not have trailing space from empty {attrs}
		trimmed := strings.TrimSpace(output)
		if strings.HasSuffix(trimmed, " ") {
			t.Error("Output should not have trailing space from empty {attrs}")
		}

		// Should be "INFO test message" without extra spaces
		expected := "INFO test message"
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain %q, got %q", expected, output)
		}
	})
}
