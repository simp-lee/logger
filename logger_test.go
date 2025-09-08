package logger

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	logger, err := New()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	if logger == nil {
		t.Fatal("Expected non-nil logger")
	}
	if logger.Logger == nil {
		t.Fatal("Expected non-nil embedded slog.Logger")
	}
	defer logger.Close()
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

	// Create a test logger
	logger, err := New(WithLevel(slog.LevelDebug))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Set as default
	logger.SetDefault()

	// Test that the default was set (this is a basic test since we can't easily capture the output)
	if slog.Default() == originalDefault {
		t.Error("Default logger was not changed")
	}
}

func TestLoggerMethods(t *testing.T) {
	// Create a test logger with a buffer
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := &Logger{
		Logger: slog.New(handler),
	}

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
	logger, err := New(
		WithLevel(slog.LevelDebug),
		WithFileFormat(FormatCustom),
		WithFileFormatter("{time} [{level}] {message} {attrs}"),
		WithFilePath(logPath),
		WithConsole(false),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

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

func TestLoggerResourceManagement(t *testing.T) {
	t.Run("LoggerWithClose", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		log, err := New(
			WithFilePath(logPath),
			WithMaxSizeMB(1),
			WithRetentionDays(1),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		log.Info("Test message")

		if err := log.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}

		// Multiple closes should be safe
		if err := log.Close(); err != nil {
			t.Errorf("Second Close() returned error: %v", err)
		}
	})

	t.Run("ConsoleOnlyLogger", func(t *testing.T) {
		log, err := New(WithFile(false))
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		log.Info("Console only message")

		if err := log.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("HandlerWithCloserAPI", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		// Create handler with closer
		result, err := newHandler(
			WithFilePath(logPath),
			WithMaxSizeMB(1),
			WithRetentionDays(1),
		)
		if err != nil {
			t.Fatalf("Failed to create handler with closer: %v", err)
		}

		// Create logger
		log := &Logger{
			Logger: slog.New(result.handler),
			closer: result.closer,
		}

		// Use the logger
		log.Info("Test message")

		// Close should work
		if err := log.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}
	})

	t.Run("RotatingWriterResourceCleanup", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &rotatingConfig{
			directory:     tmpDir,
			fileName:      "test.log",
			maxSizeMB:     1,
			retentionDays: 1,
		}

		// Create rotating writer directly
		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}

		// Write some data
		_, err = writer.Write([]byte("test data\n"))
		if err != nil {
			t.Errorf("Failed to write: %v", err)
		}

		// Close should clean up resources
		if err := writer.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}

		// Verify log file was created
		logPath := filepath.Join(tmpDir, "test.log")
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("Log file was not created")
		}
	})

	t.Run("MultipleHandlersWithResources", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "multi.log")

		log, err := New(
			WithConsole(true),
			WithFilePath(logPath),
			WithFileFormat(FormatJSON),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		log.Info("Test message to both console and file")

		if err := log.Close(); err != nil {
			t.Errorf("Close() returned error: %v", err)
		}

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Error("Log file was not created")
		}
	})
}

func TestResourceLeakPrevention(t *testing.T) {
	t.Run("GoroutineCleanup", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "goroutine_test.log")

		// Create multiple loggers to ensure goroutines are cleaned up
		for i := 0; i < 5; i++ {
			log, err := New(
				WithFilePath(logPath),
				WithMaxSizeMB(1),
			)
			if err != nil {
				t.Fatalf("Failed to create logger %d: %v", i, err)
			}

			log.Info("Test message", "iteration", i)

			if err := log.Close(); err != nil {
				t.Errorf("Failed to close logger %d: %v", i, err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	})

	t.Run("TimerCleanup", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "timer_test.log")

		log, err := New(
			WithFilePath(logPath),
			WithRetentionDays(1),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		log.Info("Test message")

		if err := log.Close(); err != nil {
			t.Errorf("Failed to close logger: %v", err)
		}

		time.Sleep(100 * time.Millisecond)
	})
}
