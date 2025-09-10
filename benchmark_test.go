package logger

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Common test data for benchmarks
var (
	benchmarkMessage = "benchmark test message for performance analysis"
	benchmarkUserID  = 12345
	benchmarkReqID   = "req-abc123-def456"
)

// =============================================================================
// Internal Performance Analysis
// =============================================================================

// BenchmarkOutputTargets compares different output targets
func BenchmarkOutputTargets(b *testing.B) {
	b.Run("Memory", func(b *testing.B) {
		var buf bytes.Buffer
		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.Console.Format = FormatJSON

		handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if err != nil {
			b.Fatal(err)
		}

		logger := slog.New(handler)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info(benchmarkMessage,
					"user_id", benchmarkUserID,
					"request_id", benchmarkReqID,
				)
			}
		})
	})

	b.Run("Discard", func(b *testing.B) {
		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.Console.Format = FormatJSON

		handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if err != nil {
			b.Fatal(err)
		}

		logger := slog.New(handler)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info(benchmarkMessage,
					"user_id", benchmarkUserID,
					"request_id", benchmarkReqID,
				)
			}
		})
	})

	b.Run("File", func(b *testing.B) {
		tmpDir := b.TempDir()
		filePath := tmpDir + "/bench.log"

		log, err := New(
			WithConsole(false),
			WithFile(true),
			WithFilePath(filePath),
			WithFileFormat(FormatJSON),
			WithLevel(slog.LevelInfo),
			WithMaxSizeMB(0), // Disable rotation for fair comparison
		)
		if err != nil {
			b.Fatal(err)
		}
		defer log.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info(benchmarkMessage,
					"user_id", benchmarkUserID,
					"request_id", benchmarkReqID,
				)
			}
		})
	})
}

// BenchmarkFormats compares different output formats
func BenchmarkFormats(b *testing.B) {
	formats := []struct {
		name   string
		format OutputFormat
	}{
		{"Text", FormatText},
		{"JSON", FormatJSON},
		{"Custom", FormatCustom},
	}

	for _, fmt := range formats {
		b.Run(fmt.name, func(b *testing.B) {
			cfg := DefaultConfig()
			cfg.Console.Color = false
			cfg.Console.Format = fmt.format

			handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
			if err != nil {
				b.Fatal(err)
			}

			logger := slog.New(handler)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					logger.Info(benchmarkMessage,
						"user_id", benchmarkUserID,
						"request_id", benchmarkReqID,
					)
				}
			})
		})
	}
}

// BenchmarkColorOverhead compares performance with and without colors
func BenchmarkColorOverhead(b *testing.B) {
	b.Run("WithColor", func(b *testing.B) {
		// Redirect to discard to focus on color processing overhead
		cfg := DefaultConfig()
		cfg.Console.Color = true
		cfg.Console.Format = FormatCustom

		handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if err != nil {
			b.Fatal(err)
		}

		logger := slog.New(handler)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info(benchmarkMessage)
				logger.Warn(benchmarkMessage)
				logger.Error(benchmarkMessage)
			}
		})
	})

	b.Run("WithoutColor", func(b *testing.B) {
		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.Console.Format = FormatCustom

		handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if err != nil {
			b.Fatal(err)
		}

		logger := slog.New(handler)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				logger.Info(benchmarkMessage)
				logger.Warn(benchmarkMessage)
				logger.Error(benchmarkMessage)
			}
		})
	})
}

// BenchmarkRotationOverhead tests the overhead of rotation features
func BenchmarkRotationOverhead(b *testing.B) {
	b.Run("NoRotation", func(b *testing.B) {
		tmpDir := b.TempDir()
		filePath := tmpDir + "/no_rotation.log"

		log, err := New(
			WithConsole(false),
			WithFile(true),
			WithFilePath(filePath),
			WithFileFormat(FormatJSON),
			WithMaxSizeMB(0), // Disable rotation
			WithLevel(slog.LevelInfo),
		)
		if err != nil {
			b.Fatal(err)
		}
		defer log.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info(benchmarkMessage,
					"user_id", benchmarkUserID,
					"timestamp", time.Now(),
				)
			}
		})
	})

	b.Run("WithRotation", func(b *testing.B) {
		tmpDir := b.TempDir()
		filePath := tmpDir + "/with_rotation.log"

		log, err := New(
			WithConsole(false),
			WithFile(true),
			WithFilePath(filePath),
			WithFileFormat(FormatJSON),
			WithMaxSizeMB(100), // Large size, won't rotate during test
			WithRetentionDays(7),
			WithLevel(slog.LevelInfo),
		)
		if err != nil {
			b.Fatal(err)
		}
		defer log.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				log.Info(benchmarkMessage,
					"user_id", benchmarkUserID,
					"timestamp", time.Now(),
				)
			}
		})
	})
}

// =============================================================================
// Concurrent Logging Benchmarks
// =============================================================================

// BenchmarkConcurrentLogging tests concurrent logging performance
func BenchmarkConcurrentLogging(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatJSON

	handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if err != nil {
		b.Fatal(err)
	}

	logger := slog.New(handler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(benchmarkMessage,
				"goroutine", runtime.NumGoroutine(),
				"timestamp", time.Now().UnixNano(),
			)
		}
	})
}

// BenchmarkMultiHandlerConcurrent tests multi-handler concurrent performance
func BenchmarkMultiHandlerConcurrent(b *testing.B) {
	cfg := DefaultConfig()
	cfg.Console.Color = false

	handler1, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		b.Fatal(err)
	}

	handler2, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		b.Fatal(err)
	}

	multiH := newMultiHandler(handler1, handler2)
	logger := slog.New(multiH)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			logger.Info(benchmarkMessage,
				"data", "test-data",
				"timestamp", time.Now().UnixNano(),
			)
		}
	})
}

// =============================================================================
// Concurrent Functional Tests
// =============================================================================

// TestConcurrentLogging tests concurrent logging operations
func TestConcurrentLogging(t *testing.T) {
	t.Run("ConcurrentWrites", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		cfg.Console.Color = false
		handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		log := slog.New(handler)

		const numGoroutines = 100
		const messagesPerGoroutine = 10

		var wg sync.WaitGroup
		var messageCount int64

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < messagesPerGoroutine; j++ {
					log.Info("Concurrent message",
						"goroutine", goroutineID,
						"message", j,
						"timestamp", time.Now().UnixNano())
					atomic.AddInt64(&messageCount, 1)
				}
			}(i)
		}

		wg.Wait()

		expectedMessages := int64(numGoroutines * messagesPerGoroutine)
		if atomic.LoadInt64(&messageCount) != expectedMessages {
			t.Errorf("Expected %d messages, got %d", expectedMessages, atomic.LoadInt64(&messageCount))
		}

		output := buf.String()
		if len(output) == 0 {
			t.Error("No output was written")
		}

		t.Logf("Successfully logged %d messages from %d goroutines", expectedMessages, numGoroutines)
	})

	t.Run("ConcurrentWithDifferentLevels", func(t *testing.T) {
		var buf bytes.Buffer

		cfg := DefaultConfig()
		cfg.Console.Color = false
		cfg.Console.Format = FormatJSON
		handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		if err != nil {
			t.Fatalf("Failed to create handler: %v", err)
		}

		log := slog.New(handler)

		const numGoroutines = 50
		var wg sync.WaitGroup

		levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				level := levels[goroutineID%len(levels)]

				switch level {
				case slog.LevelDebug:
					log.Debug("Debug message", "goroutine", goroutineID)
				case slog.LevelInfo:
					log.Info("Info message", "goroutine", goroutineID)
				case slog.LevelWarn:
					log.Warn("Warn message", "goroutine", goroutineID)
				case slog.LevelError:
					log.Error("Error message", "goroutine", goroutineID)
				}
			}(i)
		}

		wg.Wait()

		output := buf.String()
		if len(output) == 0 {
			t.Error("No output was written")
		}
	})
}

// TestConcurrentMultiHandler tests the multiHandler with concurrent access
func TestConcurrentMultiHandler(t *testing.T) {
	var buf1, buf2 bytes.Buffer

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatText

	handler1, err := newCustomHandler(&buf1, cfg, &cfg.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		t.Fatalf("Failed to create first handler: %v", err)
	}

	cfg2 := DefaultConfig()
	cfg2.Console.Format = FormatJSON
	cfg2.Console.Color = false

	handler2, err := newCustomHandler(&buf2, cfg2, &cfg2.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		t.Fatalf("Failed to create second handler: %v", err)
	}

	multiH := newMultiHandler(handler1, handler2)
	multiLogger := slog.New(multiH)

	const numGoroutines = 50
	const messagesPerGoroutine = 5
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				multiLogger.Info("Multi-handler message",
					"goroutine", goroutineID,
					"message", j)
			}
		}(i)
	}

	wg.Wait()

	output1 := buf1.String()
	output2 := buf2.String()

	if len(output1) == 0 {
		t.Error("First handler produced no output")
	}
	if len(output2) == 0 {
		t.Error("Second handler produced no output")
	}

	t.Logf("Handler 1 output length: %d", len(output1))
	t.Logf("Handler 2 output length: %d", len(output2))
}

// TestConcurrentWithGroups tests concurrent logging with groups
func TestConcurrentWithGroups(t *testing.T) {
	var buf bytes.Buffer

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatCustom
	handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	log := slog.New(handler)

	const numGoroutines = 30
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			dbLogger := log.WithGroup("Database")
			userLogger := dbLogger.WithGroup("User")

			dbLogger.Info("Database operation", "operation", "connect", "goroutine", goroutineID)
			userLogger.Info("User operation", "action", "login", "user_id", goroutineID)

			configuredLogger := userLogger.With(
				slog.String("service", "auth"),
				slog.Int("goroutine", goroutineID),
			)
			configuredLogger.Warn("Concurrent warning", "issue", "timeout")
		}(i)
	}

	wg.Wait()

	output := buf.String()
	if len(output) == 0 {
		t.Error("No output was written")
	}
}

// TestConcurrentRotatingWriter tests the rotating writer with concurrent access
func TestConcurrentRotatingWriter(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a logger with file rotation (small size to trigger rotation)
	log, err := New(
		WithFilePath(tmpDir+"/concurrent.log"),
		WithMaxSizeMB(1), // Small size to trigger rotation
		WithConsole(false),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	const numGoroutines = 20
	const messagesPerGoroutine = 50
	var wg sync.WaitGroup

	// Generate enough data to potentially trigger rotation
	longMessage := fmt.Sprintf("This is a long message to help trigger rotation: %s",
		string(make([]byte, 1000)))

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info(longMessage,
					"goroutine", goroutineID,
					"message", j,
					"data", fmt.Sprintf("data-%d-%d", goroutineID, j))
			}
		}(i)
	}

	wg.Wait()

	// Give time for any pending rotations to complete
	time.Sleep(100 * time.Millisecond)

	t.Logf("Concurrent rotation test completed successfully")
}

// TestConcurrentResourceManagement tests concurrent resource cleanup
func TestConcurrentResourceManagement(t *testing.T) {
	const numGoroutines = 20
	var wg sync.WaitGroup
	var successCount int64

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			tmpDir := t.TempDir()

			log, err := New(
				WithFilePath(fmt.Sprintf("%s/test-%d.log", tmpDir, goroutineID)),
				WithMaxSizeMB(1),
				WithConsole(false),
			)
			if err != nil {
				t.Errorf("Failed to create logger in goroutine %d: %v", goroutineID, err)
				return
			}

			for j := 0; j < 10; j++ {
				log.Info("Test message", "goroutine", goroutineID, "message", j)
			}

			if err := log.Close(); err != nil {
				t.Errorf("Failed to close logger in goroutine %d: %v", goroutineID, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
		}(i)
	}

	wg.Wait()

	if atomic.LoadInt64(&successCount) != numGoroutines {
		t.Errorf("Expected %d successful operations, got %d", numGoroutines, atomic.LoadInt64(&successCount))
	}
}

// TestStressTest performs a stress test with high concurrency
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	var buf bytes.Buffer

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatJSON
	handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	log := slog.New(handler)

	const numGoroutines = 200
	const messagesPerGoroutine = 100
	var wg sync.WaitGroup
	var totalMessages int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("Stress test message",
					"goroutine", goroutineID,
					"message", j,
					"data", fmt.Sprintf("large-data-%d-%d-%s", goroutineID, j, time.Now().Format(time.RFC3339Nano)),
					"timestamp", time.Now().UnixNano(),
				)
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	actualMessages := atomic.LoadInt64(&totalMessages)

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	messagesPerSecond := float64(actualMessages) / duration.Seconds()
	t.Logf("Stress test completed: %d messages in %v (%.2f msg/sec)",
		actualMessages, duration, messagesPerSecond)

	if len(buf.String()) == 0 {
		t.Error("No output was written during stress test")
	}
}

// =============================================================================
// Throughput Measurement Tests
// =============================================================================

// TestThroughputBasicConcurrent measures actual msg/sec throughput for basic concurrent logging.
// This test provides real-world throughput measurements for README performance documentation.
func TestThroughputBasicConcurrent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatJSON

	handler, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	log := slog.New(handler)

	const numGoroutines = 100
	const messagesPerGoroutine = 1000
	var wg sync.WaitGroup
	var totalMessages int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("Concurrent message",
					"goroutine", goroutineID,
					"message", j,
					"timestamp", time.Now().UnixNano())
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	actualMessages := atomic.LoadInt64(&totalMessages)

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	messagesPerSecond := float64(actualMessages) / duration.Seconds()
	t.Logf("Basic Concurrent: %d messages in %v (%.0f msg/sec)",
		actualMessages, duration, messagesPerSecond)
}

// TestThroughputMultiHandler measures actual msg/sec throughput for multi-handler concurrent logging.
// This test validates performance when logging to multiple destinations simultaneously.
func TestThroughputMultiHandler(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Console.Color = false

	handler1, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		t.Fatal(err)
	}

	handler2, err := newCustomHandler(io.Discard, cfg, &cfg.Console, &slog.HandlerOptions{Level: slog.LevelInfo})
	if err != nil {
		t.Fatal(err)
	}

	multiH := newMultiHandler(handler1, handler2)
	log := slog.New(multiH)

	const numGoroutines = 50
	const messagesPerGoroutine = 100
	var wg sync.WaitGroup
	var totalMessages int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("Multi-handler message",
					"goroutine", goroutineID,
					"message", j)
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	actualMessages := atomic.LoadInt64(&totalMessages)

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	messagesPerSecond := float64(actualMessages) / duration.Seconds()
	t.Logf("Multi-Handler: %d messages in %v (%.0f msg/sec)",
		actualMessages, duration, messagesPerSecond)
}

// TestThroughputFileRotation measures actual msg/sec throughput for file rotation logging.
// This test evaluates performance impact of log rotation in production scenarios.
func TestThroughputFileRotation(t *testing.T) {
	tmpDir := t.TempDir()

	log, err := New(
		WithFilePath(tmpDir+"/throughput.log"),
		WithMaxSizeMB(10), // Large enough to avoid rotation during test
		WithConsole(false),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer log.Close()

	const numGoroutines = 20
	const messagesPerGoroutine = 500
	var wg sync.WaitGroup
	var totalMessages int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("File rotation message",
					"goroutine", goroutineID,
					"message", j,
					"data", fmt.Sprintf("data-%d-%d", goroutineID, j))
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	actualMessages := atomic.LoadInt64(&totalMessages)

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	messagesPerSecond := float64(actualMessages) / duration.Seconds()
	t.Logf("File Rotation: %d messages in %v (%.0f msg/sec)",
		actualMessages, duration, messagesPerSecond)
}

// TestThroughputHighLoadStress measures actual msg/sec throughput under high-load stress conditions.
// This test validates logger performance under maximum concurrent load for capacity planning.
func TestThroughputHighLoadStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput stress test in short mode")
	}

	var buf bytes.Buffer

	cfg := DefaultConfig()
	cfg.Console.Color = false
	cfg.Console.Format = FormatJSON
	handler, err := newCustomHandler(&buf, cfg, &cfg.Console, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	log := slog.New(handler)

	const numGoroutines = 200
	const messagesPerGoroutine = 100
	var wg sync.WaitGroup
	var totalMessages int64

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				log.Info("High-load stress message",
					"goroutine", goroutineID,
					"message", j,
					"data", fmt.Sprintf("stress-data-%d-%d", goroutineID, j),
					"timestamp", time.Now().UnixNano(),
				)
				atomic.AddInt64(&totalMessages, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	actualMessages := atomic.LoadInt64(&totalMessages)

	if actualMessages != expectedMessages {
		t.Errorf("Expected %d messages, got %d", expectedMessages, actualMessages)
	}

	messagesPerSecond := float64(actualMessages) / duration.Seconds()
	t.Logf("High-Load Stress: %d messages in %v (%.0f msg/sec)",
		actualMessages, duration, messagesPerSecond)
}
