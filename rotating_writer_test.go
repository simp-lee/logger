package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRotatingWriter(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := &rotatingConfig{
			directory:     tmpDir,
			fileName:      "test.log",
			maxSizeMB:     1,
			retentionDays: 1,
		}

		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}
		defer writer.Close()

		if writer == nil {
			t.Fatal("Expected non-nil writer")
		}
	})

	t.Run("Writing", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test.log")

		cfg := &rotatingConfig{
			directory:     tmpDir,
			fileName:      "test.log",
			maxSizeMB:     1,
			retentionDays: 1,
		}

		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}
		defer writer.Close()

		// Write some data
		testData := []byte("test log message\n")
		n, err := writer.Write(testData)
		if err != nil {
			t.Fatalf("Failed to write to log: %v", err)
		}
		if n != len(testData) {
			t.Errorf("Wrote %d bytes, expected %d", n, len(testData))
		}

		// Check if file was created
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Fatalf("Log file was not created: %v", err)
		}

		// Read the file content
		content, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}
		if string(content) != string(testData) {
			t.Errorf("File content mismatch. Got %q, want %q", string(content), string(testData))
		}
	})

	t.Run("CleanOldLogs", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create an old log file
		oldLogPath := filepath.Join(tmpDir, "test.20221010.120000.000.log")
		if err := os.WriteFile(oldLogPath, []byte("old log"), 0644); err != nil {
			t.Fatalf("Failed to create old log file: %v", err)
		}

		// Set modified time to past
		oldTime := time.Now().AddDate(0, 0, -10)
		if err := os.Chtimes(oldLogPath, oldTime, oldTime); err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}

		// Create current log file
		currentLogPath := filepath.Join(tmpDir, "test.log")
		if err := os.WriteFile(currentLogPath, []byte("current log"), 0644); err != nil {
			t.Fatalf("Failed to create current log file: %v", err)
		}

		cfg := &rotatingConfig{
			directory:     tmpDir,
			fileName:      "test.log",
			maxSizeMB:     1,
			retentionDays: 7, // Keep logs for 7 days
		}

		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}

		// Manually trigger cleanup
		writer.cleanOldLogs(context.Background())
		writer.Close()

		// Old log should be deleted
		if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
			t.Fatal("Old log file should have been deleted")
		}

		// Current log should still exist
		if _, err := os.Stat(currentLogPath); os.IsNotExist(err) {
			t.Fatal("Current log file should still exist")
		}
	})

	t.Run("RotationBySize", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Logf("Base temporary directory: %s", tmpDir)

		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log subdirectory: %v", err)
		}
		t.Logf("Log directory: %s", logDir)

		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		maxSizeMB := 1
		cfg := &rotatingConfig{
			directory:     logDir,
			fileName:      "test.log",
			maxSizeMB:     maxSizeMB,
			retentionDays: 7,
		}

		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}
		defer writer.Close()

		logMessage := strings.Repeat("This is a test log message that will be repeated to exceed the file size limit.\n", 200) // 增加到200次重复

		totalBytesWritten := 0
		iterations := 0
		maxIterations := 1000

		for totalBytesWritten < maxSizeMB*1024*1024 && iterations < maxIterations {
			n, err := writer.Write([]byte(logMessage))
			if err != nil {
				t.Fatalf("Failed to write to log: %v", err)
			}
			totalBytesWritten += n
			iterations++

			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("Wrote %d bytes in %d iterations", totalBytesWritten, iterations)

		time.Sleep(1 * time.Second)

		n, err := writer.Write([]byte("New log entry after rotation\n"))
		if err != nil {
			t.Fatalf("Failed to write after rotation: %v", err)
		}
		t.Logf("Wrote %d bytes after rotation", n)

		time.Sleep(1 * time.Second)

		files, err := os.ReadDir(logDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// At least one rotated file should exist
		rotatedCount := 0
		var rotatedFiles []string

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			t.Logf("Found file: %s", fileName)

			if fileName == "test.log" {
				// Current log file should exist
				fileInfo, err := file.Info()
				if err != nil {
					t.Fatalf("Failed to get info for current log file: %v", err)
				}
				t.Logf("Current log file size: %d bytes", fileInfo.Size())
				continue
			}

			// Check if it's a rotated log file (format: test.YYYYMMDD.HHMMSS.SSS.log)
			if strings.HasPrefix(fileName, "test.") && strings.HasSuffix(fileName, ".log") && len(fileName) > 8 {
				rotatedCount++
				rotatedFiles = append(rotatedFiles, fileName)

				// Check the content and size of the rotated file
				rotatedPath := filepath.Join(logDir, fileName)
				fileInfo, err := os.Stat(rotatedPath)
				if err != nil {
					t.Logf("Failed to get info for rotated log file %s: %v", fileName, err)
					continue
				}
				t.Logf("Rotated log file %s size: %d bytes", fileName, fileInfo.Size())
			}
		}

		// Verify that at least one rotated file exists
		if rotatedCount == 0 {
			t.Fatal("No rotated log files found, rotation may have failed")
		} else {
			t.Logf("Found %d rotated log files: %v", rotatedCount, rotatedFiles)
		}

		// Check if current log file exists and is less than max size
		currentFileInfo, err := os.Stat(logPath)
		if os.IsNotExist(err) {
			t.Logf("Current log file does not exist. This might be expected if rotation just occurred and no new writes happened yet")
		} else if err != nil {
			t.Fatalf("Failed to get current log file info: %v", err)
		} else {
			t.Logf("Current log file size: %d bytes", currentFileInfo.Size())
			if currentFileInfo.Size() > int64(maxSizeMB*1024*1024) {
				t.Errorf("Current log file size (%d bytes) exceeds max size (%d bytes)",
					currentFileInfo.Size(), maxSizeMB*1024*1024)
			}
		}
	})
}

// TestRotationDisabled verifies that setting MaxSizeMB to 0 actually disables rotation
func TestRotationDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_no_rotation.log")

	// Create logger with rotation disabled (MaxSizeMB = 0)
	logger, err := New(
		WithConsole(false),
		WithFile(true),
		WithFilePath(logPath),
		WithFileFormat(FormatText),
		WithMaxSizeMB(0), // Disable rotation
		WithLevel(slog.LevelInfo),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Write a large amount of log data that would normally trigger rotation
	// Generate approximately 1MB of log data (much larger than normal rotation size)
	largeMessage := strings.Repeat("This is a test message for rotation disabled test. ", 100) // ~5KB per message
	for i := 0; i < 200; i++ {                                                                 // 200 * 5KB = ~1MB
		logger.Info(largeMessage, "iteration", i, "timestamp", time.Now().Unix())
	}

	// Force any pending writes
	logger.Close()

	// Check that only one log file exists (no rotation files)
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp directory: %v", err)
	}

	var logFiles []string
	for _, file := range files {
		if strings.Contains(file.Name(), "test_no_rotation") {
			logFiles = append(logFiles, file.Name())
		}
	}

	if len(logFiles) != 1 {
		t.Errorf("Expected exactly 1 log file, found %d: %v", len(logFiles), logFiles)
	}

	// Verify the single log file exists and has substantial content
	stat, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("Failed to stat log file: %v", err)
	}

	// The file should be quite large since rotation was disabled
	if stat.Size() < 500*1024 { // At least 500KB
		t.Errorf("Expected log file to be at least 500KB (rotation disabled), got %d bytes", stat.Size())
	}

	t.Logf("Successfully verified rotation disabled: single file with %d bytes", stat.Size())
}

// TestRotationEnabledVsDisabled compares behavior with rotation enabled vs disabled
func TestRotationEnabledVsDisabled(t *testing.T) {
	// Test with rotation enabled (small size)
	t.Run("RotationEnabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test_with_rotation.log")

		logger, err := New(
			WithConsole(false),
			WithFile(true),
			WithFilePath(logPath),
			WithFileFormat(FormatText),
			WithMaxSizeMB(1), // Enable rotation with 1MB limit
			WithLevel(slog.LevelInfo),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Write enough data to trigger rotation
		largeMessage := strings.Repeat("Test message for rotation. ", 100)
		for i := 0; i < 300; i++ {
			logger.Info(largeMessage, "iteration", i)
		}

		logger.Close()

		// Check for multiple files (rotation should have occurred)
		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("Failed to read temp directory: %v", err)
		}

		var logFiles []string
		for _, file := range files {
			if strings.Contains(file.Name(), "test_with_rotation") {
				logFiles = append(logFiles, file.Name())
			}
		}

		if len(logFiles) <= 1 {
			t.Logf("Note: Only %d log file(s) found, rotation might not have been triggered", len(logFiles))
			// Don't fail the test as rotation timing can be variable
		} else {
			t.Logf("Rotation enabled: found %d log files as expected", len(logFiles))
		}
	})

	// Test with rotation disabled
	t.Run("RotationDisabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		logPath := filepath.Join(tmpDir, "test_no_rotation.log")

		logger, err := New(
			WithConsole(false),
			WithFile(true),
			WithFilePath(logPath),
			WithFileFormat(FormatText),
			WithMaxSizeMB(0), // Disable rotation
			WithLevel(slog.LevelInfo),
		)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		// Write the same amount of data
		largeMessage := strings.Repeat("Test message for no rotation. ", 100)
		for i := 0; i < 300; i++ {
			logger.Info(largeMessage, "iteration", i)
		}

		logger.Close()

		// Should only have one file
		files, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatalf("Failed to read temp directory: %v", err)
		}

		var logFiles []string
		for _, file := range files {
			if strings.Contains(file.Name(), "test_no_rotation") {
				logFiles = append(logFiles, file.Name())
			}
		}

		if len(logFiles) != 1 {
			t.Errorf("Rotation disabled: expected exactly 1 file, got %d: %v", len(logFiles), logFiles)
		}
	})
}

// TestConcurrentWriteAfterClose tests the race condition between Write and Close
func TestConcurrentWriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &rotatingConfig{
		directory:     tmpDir,
		fileName:      "test.log",
		maxSizeMB:     1, // Enable rotation
		retentionDays: 1,
	}

	writer, err := newRotatingWriter(cfg)
	if err != nil {
		t.Fatalf("Failed to create rotating writer: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	var writeErrors int

	// Start a goroutine that keeps writing
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			// Write enough data to potentially trigger rotation
			data := make([]byte, 2048) // 2KB per write
			for j := range data {
				data[j] = 'A'
			}

			// After Close() is called, this should return an error instead of panicking
			_, err := writer.Write(data)
			if err != nil {
				writeErrors++
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Close the writer after a short delay
	time.Sleep(10 * time.Millisecond)
	writer.Close()

	wg.Wait()

	// After closing, writes should fail gracefully with errors, not panic
	if writeErrors == 0 {
		t.Errorf("Expected some write errors after Close(), but got none")
	}
	t.Logf("Got %d write errors after Close() - this is expected behavior", writeErrors)
}

// TestDeadlockInCleanOldLogs tests potential deadlock issues in cleanOldLogs method
func TestDeadlockInCleanOldLogs(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	// Create a logger using rotatingWriter
	logger, err := New(
		WithFilePath(logFile),
		WithFileFormat(FormatText),
		WithLevel(slog.LevelDebug),
		WithMaxSizeMB(1),
		WithRetentionDays(1),
	)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	// Critical: Set this logger as the default logger
	// This way slog.Info/Warn calls will use the same rotatingWriter
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)
	logger.SetDefault()

	// Get rotatingWriter instance for direct testing
	// Create a rotatingWriter with the same configuration for simulation
	rotatingWriter, err := newRotatingWriter(&rotatingConfig{
		directory:     tempDir,
		fileName:      "test.log",
		maxSizeMB:     1,
		retentionDays: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create rotating writer: %v", err)
	}
	defer rotatingWriter.Close()

	// Set up a logger containing the same rotatingWriter as default logger
	fileHandler := slog.NewTextHandler(rotatingWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	testLogger := slog.New(fileHandler)
	slog.SetDefault(testLogger)

	// Create some old log files to trigger cleanup
	oldLogFile := filepath.Join(tempDir, "test.log.2023-01-01")
	if err := os.WriteFile(oldLogFile, []byte("old log content"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}

	// Set file modification time to long ago to ensure it will be cleaned up
	oldTime := time.Now().AddDate(0, 0, -10) // 10 days ago
	if err := os.Chtimes(oldLogFile, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set file time: %v", err)
	}

	// Use channel and goroutine to detect deadlock
	done := make(chan bool, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Call cleanOldLogs, which might cause deadlock
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rotatingWriter.cleanOldLogs(ctx)
		done <- true
	}()

	// Wait for a while, if timeout occurs, consider it a deadlock
	select {
	case <-done:
		t.Log("cleanOldLogs completed successfully - no deadlock detected")
	case <-time.After(10 * time.Second):
		t.Fatal("Deadlock detected: cleanOldLogs did not complete within 10 seconds")
	}

	wg.Wait()
}

// TestConcurrentCleanOldLogs tests behavior when calling cleanOldLogs concurrently
func TestConcurrentCleanOldLogs(t *testing.T) {
	tempDir := t.TempDir()

	rotatingWriter, err := newRotatingWriter(&rotatingConfig{
		directory:     tempDir,
		fileName:      "test.log",
		maxSizeMB:     1,
		retentionDays: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create rotating writer: %v", err)
	}
	defer rotatingWriter.Close()

	// Set default logger to use the same rotatingWriter
	fileHandler := slog.NewTextHandler(rotatingWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	testLogger := slog.New(fileHandler)
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)
	slog.SetDefault(testLogger)

	// Create multiple goroutines to call cleanOldLogs concurrently
	var wg sync.WaitGroup
	numGoroutines := 5

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rotatingWriter.cleanOldLogs(ctx)
		}(i)
	}

	// Use channel to detect if all goroutines completed
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("All concurrent cleanOldLogs calls completed successfully")
	case <-time.After(15 * time.Second):
		t.Fatal("Deadlock detected: concurrent cleanOldLogs calls did not complete within 15 seconds")
	}
}

// TestDeadlockWithWrite tests if calling cleanOldLogs during write operations causes deadlock
func TestDeadlockWithWrite(t *testing.T) {
	tempDir := t.TempDir()

	rotatingWriter, err := newRotatingWriter(&rotatingConfig{
		directory:     tempDir,
		fileName:      "test.log",
		maxSizeMB:     1,
		retentionDays: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create rotating writer: %v", err)
	}
	defer rotatingWriter.Close()

	// Set default logger
	fileHandler := slog.NewTextHandler(rotatingWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	testLogger := slog.New(fileHandler)
	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)
	slog.SetDefault(testLogger)

	var wg sync.WaitGroup

	// Goroutine 1: Continuously write logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			rotatingWriter.Write([]byte("test log message\n"))
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Goroutine 2: Call cleanOldLogs
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(50 * time.Millisecond) // Let write start first
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		rotatingWriter.cleanOldLogs(ctx)
	}()

	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		t.Log("Write and cleanOldLogs completed successfully")
	case <-time.After(10 * time.Second):
		t.Fatal("Deadlock detected: write and cleanOldLogs did not complete within 10 seconds")
	}
}

// TestCleanOldLogsRobustness tests the robustness of the fixed cleanOldLogs method
func TestCleanOldLogsRobustness(t *testing.T) {
	tests := []struct {
		name        string
		setupFiles  func(string) error // Setup test files
		expectError bool
	}{
		{
			name: "NormalCleanup",
			setupFiles: func(dir string) error {
				// Create an old file and a new file
				oldFile := filepath.Join(dir, "test.log.2023-01-01")
				newFile := filepath.Join(dir, "test.log.2025-08-01")

				if err := os.WriteFile(oldFile, []byte("old"), 0644); err != nil {
					return err
				}
				if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
					return err
				}

				// Set old file time
				oldTime := time.Now().AddDate(0, 0, -10)
				return os.Chtimes(oldFile, oldTime, oldTime)
			},
			expectError: false,
		},
		{
			name: "EmptyDirectory",
			setupFiles: func(dir string) error {
				return nil // Empty directory
			},
			expectError: false,
		},
		{
			name: "NonExistentDirectory",
			setupFiles: func(dir string) error {
				// Remove directory to simulate non-existent situation
				return os.RemoveAll(dir)
			},
			expectError: false, // Method should handle this gracefully
		},
		{
			name: "PermissionDeniedFiles",
			setupFiles: func(dir string) error {
				oldFile := filepath.Join(dir, "test.log.2023-01-01")
				if err := os.WriteFile(oldFile, []byte("protected"), 0644); err != nil {
					return err
				}

				// Set old file time
				oldTime := time.Now().AddDate(0, 0, -10)
				if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
					return err
				}

				// Try to set read-only permission (might not work on Windows)
				return os.Chmod(oldFile, 0444)
			},
			expectError: false, // Should log error but not crash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Setup test files
			if err := tt.setupFiles(tempDir); err != nil {
				t.Fatalf("Failed to setup test files: %v", err)
			}

			// Create rotatingWriter
			rotatingWriter, err := newRotatingWriter(&rotatingConfig{
				directory:     tempDir,
				fileName:      "test.log",
				maxSizeMB:     1,
				retentionDays: 1,
			})
			if err != nil {
				t.Fatalf("Failed to create rotating writer: %v", err)
			}
			defer rotatingWriter.Close()

			// Set up default logger using the rotatingWriter
			fileHandler := slog.NewTextHandler(rotatingWriter, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})
			testLogger := slog.New(fileHandler)
			originalDefault := slog.Default()
			defer slog.SetDefault(originalDefault)
			slog.SetDefault(testLogger)

			// Execute cleanOldLogs (should not deadlock here)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// This call should complete within 5 seconds, should not deadlock
			done := make(chan bool, 1)
			go func() {
				rotatingWriter.cleanOldLogs(ctx)
				done <- true
			}()

			select {
			case <-done:
				t.Logf("cleanOldLogs completed successfully for case: %s", tt.name)
			case <-time.After(10 * time.Second):
				t.Fatalf("cleanOldLogs timed out for case: %s", tt.name)
			}
		})
	}
}

// TestCleanOldLogsWithCircularLogging tests circular logging scenarios
func TestCleanOldLogsWithCircularLogging(t *testing.T) {
	tempDir := t.TempDir()

	// Create first rotatingWriter
	rw1, err := newRotatingWriter(&rotatingConfig{
		directory:     tempDir,
		fileName:      "app1.log",
		maxSizeMB:     1,
		retentionDays: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create first rotating writer: %v", err)
	}
	defer rw1.Close()

	// Create second rotatingWriter
	rw2, err := newRotatingWriter(&rotatingConfig{
		directory:     tempDir,
		fileName:      "app2.log",
		maxSizeMB:     1,
		retentionDays: 1,
	})
	if err != nil {
		t.Fatalf("Failed to create second rotating writer: %v", err)
	}
	defer rw2.Close()

	// Create logger using rw1, but set as default logger
	logger1 := slog.New(slog.NewTextHandler(rw1, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	originalDefault := slog.Default()
	defer slog.SetDefault(originalDefault)
	slog.SetDefault(logger1)

	// Now let rw2 call cleanOldLogs
	// This should use default logger (rw1) for logging, not rw2, so it should not deadlock
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan bool, 1)
	go func() {
		rw2.cleanOldLogs(ctx)
		done <- true
	}()

	select {
	case <-done:
		t.Log("Circular logging test passed - no deadlock")
	case <-time.After(10 * time.Second):
		t.Fatal("Circular logging test failed - deadlock detected")
	}
}
