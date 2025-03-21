package logger

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	handler := slog.NewTextHandler(io.Discard, nil)
	logger := New(handler)
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
	if logger.Logger == nil {
		t.Fatal("Expected non-nil embedded slog.Logger")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
	if logger.Logger == nil {
		t.Fatal("Expected non-nil embedded slog.Logger")
	}
}

func TestSetDefault(t *testing.T) {
	// Save the original default logger
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)

	// Create a test logger with a buffer
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := New(handler)

	// Set as default
	logger.SetDefault()

	// Use the standard slog methods
	slog.Info("test message")

	// Check if the message was written to our buffer
	if !strings.Contains(buf.String(), "test message") {
		t.Errorf("Default logger was not set correctly, message not found in buffer")
	}
}

func TestLoggerMethods(t *testing.T) {
	// Create a test logger with a buffer
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := New(handler)

	// Test various logging methods
	logger.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Errorf("Debug message not logged")
	}
	buf.Reset()

	logger.Info("info message")
	if !strings.Contains(buf.String(), "info message") {
		t.Errorf("Info message not logged")
	}
	buf.Reset()

	logger.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("Warn message not logged")
	}
	buf.Reset()

	logger.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("Error message not logged")
	}
	buf.Reset()

	// Test With methods
	logger.With("key", "value").Info("with message")
	if !strings.Contains(buf.String(), "with message") || !strings.Contains(buf.String(), "key") || !strings.Contains(buf.String(), "value") {
		t.Errorf("With attribute not included in log")
	}
}

func TestCustomHandlerIntegration(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "logger_test_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	logPath := tmpFile.Name()

	// Create a logger with custom handler
	handler, err := NewHandler(
		WithLevel(slog.LevelDebug),
		WithFileFormat(FormatCustom),
		WithFileFormatter("{time} [{level}] {message} {attrs}"),
		WithFilePath(logPath),
		WithConsole(false),
	)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	logger := New(handler)

	// Test logging
	logger.Info("test message", "key", "value")

	// Read the log content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "test message") {
		t.Errorf("Log message not found in file")
	}
	if !strings.Contains(logContent, "key") || !strings.Contains(logContent, "value") {
		t.Errorf("Log attributes not found in file")
	}
}
