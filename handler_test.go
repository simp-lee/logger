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
			WithFormat(FormatText),
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
			WithFormat(FormatJSON),
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
			WithFormat(FormatCustom),
			WithFormatter("{time} [{level}] {message}"),
		)
		if err != nil {
			t.Fatalf("Failed to create custom console handler: %v", err)
		}
		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})

	t.Run("FileHandler", func(t *testing.T) {
		// 创建临时目录
		tmpDir := t.TempDir()

		// 调试信息，帮助排查路径问题
		t.Logf("Temporary directory: %s", tmpDir)

		// 手动创建 "001" 目录
		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log directory: %v", err)
		}

		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithFormat(FormatText),
			WithFile(true),
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

		// 打印一条日志以确保文件被创建
		logger := New(handler)
		logger.Info("Test log message")

		// 检查文件是否被创建
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatalf("Log file was not created: %v", err)
		} else if err != nil {
			t.Fatalf("Error checking log file: %v", err)
		}
	})

	t.Run("MultiHandler", func(t *testing.T) {
		// 创建临时目录
		tmpDir := t.TempDir()

		// 调试信息，帮助排查路径问题
		t.Logf("Temporary directory: %s", tmpDir)

		// 手动创建 "001" 目录
		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log directory: %v", err)
		}

		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		handler, err := NewHandler(
			WithLevel(slog.LevelInfo),
			WithFormat(FormatJSON),
			WithFile(true),
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

		// 打印一条日志以确保文件被创建
		logger := New(handler)
		logger.Info("Test log message")

		// 检查文件是否被创建
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatalf("Log file was not created: %v", err)
		} else if err != nil {
			t.Fatalf("Error checking log file: %v", err)
		}
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		_, err := NewHandler(
			WithFormat("invalid"),
		)
		if err == nil {
			t.Fatal("Expected error for invalid format")
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

func TestConsoleHandler(t *testing.T) {
	writer := &mockWriter{}

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	// Test JSON handler
	cfg := DefaultConfig()
	cfg.Format = FormatJSON

	handler, err := newCustomHandler(writer, cfg, opts)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	if len(writer.written) == 0 {
		t.Fatal("Expected output, got none")
	}
}
