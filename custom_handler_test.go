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

	t.Run("WithGroupBasic", func(t *testing.T) {
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
		record.AddAttrs(slog.String("key", "value"))

		err = newHandler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		// With proper slog behavior, message should not contain group prefix
		if !strings.Contains(output, "test message") {
			t.Errorf("Output doesn't contain original message: %q", output)
		}
		// But attributes should be prefixed with group
		if !strings.Contains(output, "test_group.key=value") {
			t.Errorf("Output doesn't contain grouped attribute: %q", output)
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

// Test that verifies ReplaceAttr is called for built-in attributes (time, level, source, message)
// This ensures compatibility with slog.HandlerOptions.ReplaceAttr specification
func TestCustomHandler_ReplaceAttr_BuiltIns(t *testing.T) {
	var buf bytes.Buffer

	// Track which attributes are processed by ReplaceAttr
	processedAttrs := make(map[string]bool)

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Formatter = "{time} {level} {file} {message} {attrs}"
	cfg.AddSource = true
	cfg.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		processedAttrs[a.Key] = true
		// This should be called for ALL attributes including built-ins
		switch a.Key {
		case slog.TimeKey:
			return slog.String(slog.TimeKey, "CUSTOM_TIME")
		case slog.LevelKey:
			return slog.String(slog.LevelKey, "CUSTOM_LEVEL")
		case slog.SourceKey:
			return slog.String(slog.SourceKey, "CUSTOM_SOURCE")
		case slog.MessageKey:
			return slog.String(slog.MessageKey, "CUSTOM_MESSAGE")
		case "password":
			return slog.String("password", "***")
		default:
			return a
		}
	}

	outputCfg := &mockOutputConfig{
		format:    FormatCustom,
		color:     false,
		formatter: "{time} {level} {file} {message} {attrs}",
	}

	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   true,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
		PC:      1, // Set a non-zero PC to trigger source handling
	}
	record.AddAttrs(
		slog.String("password", "secret123"),
		slog.String("user", "john"),
	)

	err = handler.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	output := buf.String()
	t.Logf("Output: %q", output)

	// Verify that ReplaceAttr was called for ALL attributes (built-ins and user)
	expectedAttrs := []string{slog.TimeKey, slog.LevelKey, slog.SourceKey, slog.MessageKey, "password", "user"}
	for _, attr := range expectedAttrs {
		if !processedAttrs[attr] {
			t.Errorf("ReplaceAttr was not called for '%s' attribute", attr)
		}
	}

	// Verify that the output contains the replaced values
	if !strings.Contains(output, "CUSTOM_TIME") {
		t.Error("Output should contain replaced time value 'CUSTOM_TIME'")
	}
	if !strings.Contains(output, "CUSTOM_LEVEL") {
		t.Error("Output should contain replaced level value 'CUSTOM_LEVEL'")
	}
	if !strings.Contains(output, "CUSTOM_SOURCE") {
		t.Error("Output should contain replaced source value 'CUSTOM_SOURCE'")
	}
	if !strings.Contains(output, "CUSTOM_MESSAGE") {
		t.Error("Output should contain replaced message value 'CUSTOM_MESSAGE'")
	}
	if !strings.Contains(output, "password=***") {
		t.Error("Password should be masked")
	}
	if strings.Contains(output, "secret123") {
		t.Error("Original password should not appear")
	}
}

// Test that verifies ReplaceAttr can remove built-in attributes
func TestCustomHandler_ReplaceAttr_RemoveBuiltIns(t *testing.T) {
	var buf bytes.Buffer

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Formatter = "{time} {level} {file} {message} {attrs}"
	cfg.AddSource = true
	cfg.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
		// Remove time and source attributes
		switch a.Key {
		case slog.TimeKey, slog.SourceKey:
			return slog.Attr{} // Return empty attr to remove
		default:
			return a
		}
	}

	outputCfg := &mockOutputConfig{
		format:    FormatCustom,
		color:     false,
		formatter: "{time} {level} {file} {message} {attrs}",
	}

	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   true,
		ReplaceAttr: cfg.ReplaceAttr,
	}

	handler, err := newCustomHandler(&buf, cfg, outputCfg, opts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	record := slog.Record{
		Time:    time.Now(),
		Level:   slog.LevelInfo,
		Message: "test message",
		PC:      1,
	}
	record.AddAttrs(slog.String("user", "john"))

	err = handler.Handle(context.Background(), record)
	if err != nil {
		t.Fatalf("Handler.Handle failed: %v", err)
	}

	output := buf.String()
	t.Logf("Output: %q", output)

	// Level and message should remain, time and source should be removed
	if !strings.Contains(output, "INFO") {
		t.Error("Level should still be present")
	}
	if !strings.Contains(output, "test message") {
		t.Error("Message should still be present")
	}
	if !strings.Contains(output, "user=john") {
		t.Error("User attribute should still be present")
	}

	// The placeholders for time and file should be properly removed without leaving extra spaces
	if strings.Contains(output, "  ") {
		t.Error("Should not have double spaces from removed placeholders")
	}
}

// Test to verify our behavior matches standard slog handlers
func TestCustomHandler_StandardSlogCompatibility(t *testing.T) {
	// Test that our ReplaceAttr behavior matches standard slog.TextHandler
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.TimeKey:
			return slog.String(slog.TimeKey, "CUSTOM_TIME")
		case slog.LevelKey:
			return slog.String(slog.LevelKey, "CUSTOM_LEVEL")
		case slog.MessageKey:
			return slog.String(slog.MessageKey, "CUSTOM_MESSAGE")
		case "password":
			return slog.String("password", "***")
		default:
			return a
		}
	}

	// Track processed attributes for both handlers
	standardProcessed := make(map[string]bool)
	customProcessed := make(map[string]bool)

	t.Run("StandardTextHandler", func(t *testing.T) {
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: false,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				standardProcessed[a.Key] = true
				return replaceAttr(groups, a)
			},
		})

		logger := slog.New(handler)
		logger.Info("test message", "password", "secret123", "user", "john")
	})

	t.Run("CustomHandler", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.AddSource = false
		cfg.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			customProcessed[a.Key] = true
			return replaceAttr(groups, a)
		}

		outputCfg := &mockOutputConfig{
			format:    FormatCustom,
			color:     false,
			formatter: "{time} {level} {message} {attrs}",
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
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "test message",
		}
		record.AddAttrs(
			slog.String("password", "secret123"),
			slog.String("user", "john"),
		)

		err = handler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}
	})

	// Verify both handlers processed the same set of attributes
	for attr := range standardProcessed {
		if !customProcessed[attr] {
			t.Errorf("Custom handler did not process attribute '%s' that standard handler processed", attr)
		}
	}
	for attr := range customProcessed {
		if !standardProcessed[attr] {
			t.Errorf("Custom handler processed attribute '%s' that standard handler did not process", attr)
		}
	}
}

// Test to compare behavior with standard slog handlers
func TestCustomHandler_CompareWithStandardSlog(t *testing.T) {
	// Track which attributes are processed by ReplaceAttr
	processedAttrs := make(map[string]bool)

	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		processedAttrs[a.Key] = true
		switch a.Key {
		case slog.TimeKey:
			return slog.String(slog.TimeKey, "CUSTOM_TIME")
		case slog.LevelKey:
			return slog.String(slog.LevelKey, "CUSTOM_LEVEL")
		case slog.MessageKey:
			return slog.String(slog.MessageKey, "CUSTOM_MESSAGE")
		case "password":
			return slog.String("password", "***")
		default:
			return a
		}
	}

	// Test with standard TextHandler
	t.Run("StandardTextHandler", func(t *testing.T) {
		var buf bytes.Buffer
		processedAttrs = make(map[string]bool) // Reset

		handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			AddSource:   false, // Disable to simplify comparison
			ReplaceAttr: replaceAttr,
		})

		logger := slog.New(handler)
		logger.Info("test message", "password", "secret123", "user", "john")

		output := buf.String()
		t.Logf("Standard TextHandler output: %q", output)
		t.Logf("Standard TextHandler processed attributes: %v", processedAttrs)

		// Verify all built-in attributes were processed
		expectedAttrs := []string{slog.TimeKey, slog.LevelKey, slog.MessageKey}
		for _, attr := range expectedAttrs {
			if !processedAttrs[attr] {
				t.Errorf("Standard handler: ReplaceAttr was not called for %s", attr)
			}
		}
	})

	// Test with our custom handler
	t.Run("CustomHandler", func(t *testing.T) {
		var buf bytes.Buffer
		processedAttrs = make(map[string]bool) // Reset

		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.Console.Formatter = "{time} {level} {message} {attrs}"
		cfg.AddSource = false // Disable to simplify comparison
		cfg.ReplaceAttr = replaceAttr

		outputCfg := &ConsoleConfig{
			Enabled:   true,
			Color:     false,
			Format:    FormatCustom,
			Formatter: "{time} {level} {message} {attrs}",
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
			Time:    time.Now(),
			Level:   slog.LevelInfo,
			Message: "test message",
		}
		record.AddAttrs(
			slog.String("password", "secret123"),
			slog.String("user", "john"),
		)

		err = handler.Handle(context.Background(), record)
		if err != nil {
			t.Fatalf("Handler.Handle failed: %v", err)
		}

		output := buf.String()
		t.Logf("Custom handler output: %q", output)
		t.Logf("Custom handler processed attributes: %v", processedAttrs)

		// Verify all built-in attributes were processed (same as standard)
		expectedAttrs := []string{slog.TimeKey, slog.LevelKey, slog.MessageKey}
		for _, attr := range expectedAttrs {
			if !processedAttrs[attr] {
				t.Errorf("Custom handler: ReplaceAttr was not called for %s", attr)
			}
		}

		// Verify the replacements worked
		if !strings.Contains(output, "CUSTOM_TIME") {
			t.Error("Custom handler: Output should contain replaced time value")
		}
		if !strings.Contains(output, "CUSTOM_LEVEL") {
			t.Error("Custom handler: Output should contain replaced level value")
		}
		if !strings.Contains(output, "CUSTOM_MESSAGE") {
			t.Error("Custom handler: Output should contain replaced message value")
		}
		if !strings.Contains(output, "password=***") {
			t.Error("Custom handler: Password should be masked")
		}
		if strings.Contains(output, "secret123") {
			t.Error("Custom handler: Original password should not appear")
		}
	})
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
