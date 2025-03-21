package logger

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewHandler(t *testing.T) {
	t.Run("DefaultConsoleHandler", func(t *testing.T) {
		handler, err := NewHandler()
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})

	t.Run("ConsoleHandler", func(t *testing.T) {
		handler, err := NewHandler(
			WithLevel(slog.LevelDebug),
			WithConsoleFormat(FormatText),
		)
		if err != nil {
			t.Fatalf("Failed to create console handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})

	t.Run("JSONConsoleHandler", func(t *testing.T) {
		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithConsoleFormat(FormatJSON),
		)
		if err != nil {
			t.Fatalf("Failed to create JSON console handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})

	t.Run("CustomConsoleHandler", func(t *testing.T) {
		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithConsoleFormatter("{time} [{level}] {message}"),
		)
		if err != nil {
			t.Fatalf("Failed to create custom console handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})

	t.Run("FileHandler", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Debug information to help with path issues
		t.Logf("Temporary directory: %s", tmpDir)

		// Manually create the "001" directory
		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log directory: %v", err)
		}

		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithFileFormat(FormatText),
			WithFilePath(logPath),
			WithConsole(false),
			WithMaxSizeMB(10),
			WithRetentionDays(7),
		)
		if err != nil {
			t.Fatalf("Failed to create file handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}

		// Print one log message to ensure the file is created
		logger := New(handler)
		logger.Info("Test log message")

		// Check if the file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatalf("Log file was not created: %v", err)
		} else if err != nil {
			t.Fatalf("Error checking log file: %v", err)
		}
	})

	t.Run("MultiHandler", func(t *testing.T) {
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Debug information to help with path issues
		t.Logf("Temporary directory: %s", tmpDir)

		// Manually create the "001" directory
		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log directory: %v", err)
		}

		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithConsoleFormat(FormatJSON),
			WithFileFormat(FormatJSON),
			WithFilePath(logPath),
			WithConsole(true),
			WithMaxSizeMB(10),
			WithRetentionDays(7),
		)
		if err != nil {
			t.Fatalf("Failed to create multi handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}

		// Print one log message to ensure the file is created
		logger := New(handler)
		logger.Info("Test log message")

		// Check if the file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatalf("Log file was not created: %v", err)
		} else if err != nil {
			t.Fatalf("Error checking log file: %v", err)
		}
	})

	t.Run("InvalidConsoleFormat", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Console.Format = "invalid"

		_, err := newConsoleHandler(cfg)
		if err == nil {
			t.Fatal("Expected error for invalid console format")
		}
	})

	t.Run("InvalidFileFormat", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.File.Format = "invalid"
		cfg.File.Enabled = true
		cfg.File.Path = "test.log"

		_, err := newFileHandler(cfg)
		if err == nil {
			t.Fatal("Expected error for invalid file format")
		}
	})
}

// Mock Writer for testing
type mockWriter struct {
	written []byte
}

func (m *mockWriter) Write(p []byte) (int, error) {
	m.written = append(m.written, p...)
	return len(p), nil
}

func TestHandlerWithCustomWriter(t *testing.T) {
	writer := &mockWriter{}

	cfg := DefaultConfig()

	// Test console handler with custom writer
	handler, err := newCustomHandler(writer, cfg, &cfg.Console, &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	if len(writer.written) == 0 {
		t.Fatal("Expected output, got none")
	}
}
