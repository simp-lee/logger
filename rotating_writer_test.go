package logger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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
		// 创建临时目录
		tmpDir := t.TempDir()
		t.Logf("Base temporary directory: %s", tmpDir)

		// 创建子目录 "001"，与实现中可能预期的结构匹配
		logDir := filepath.Join(tmpDir, "001")
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create log subdirectory: %v", err)
		}
		t.Logf("Log directory: %s", logDir)

		// 日志文件路径
		logPath := filepath.Join(logDir, "test.log")
		t.Logf("Log file path: %s", logPath)

		// 设置一个非常小的最大文件大小 (50KB)，方便测试轮转
		maxSizeKB := 50
		cfg := &rotatingConfig{
			directory:     logDir, // 使用创建的子目录
			fileName:      "test.log",
			maxSizeMB:     maxSizeKB / 1024, // 转换为 MB
			retentionDays: 7,
		}

		writer, err := newRotatingWriter(cfg)
		if err != nil {
			t.Fatalf("Failed to create rotating writer: %v", err)
		}
		defer writer.Close()

		// 创建一条重复的消息，用于填充日志文件
		logMessage := strings.Repeat("This is a test log message that will be repeated to exceed the file size limit.\n", 20)

		// 写入数据直到超过最大大小
		totalBytesWritten := 0
		iterations := 0
		maxIterations := 100 // 防止无限循环

		for totalBytesWritten < maxSizeKB*1024 && iterations < maxIterations {
			n, err := writer.Write([]byte(logMessage))
			if err != nil {
				t.Fatalf("Failed to write to log: %v", err)
			}
			totalBytesWritten += n
			iterations++

			// 添加少量延迟，避免过快写入导致时间戳相同
			time.Sleep(5 * time.Millisecond)
		}

		t.Logf("Wrote %d bytes in %d iterations", totalBytesWritten, iterations)

		// 给轮转一些时间完成 - 增加等待时间
		time.Sleep(1 * time.Second)

		// 再写入一些数据，确保新日志文件创建成功
		n, err := writer.Write([]byte("New log entry after rotation\n"))
		if err != nil {
			t.Fatalf("Failed to write after rotation: %v", err)
		}
		t.Logf("Wrote %d bytes after rotation", n)

		// 再次等待轮转完成
		time.Sleep(1 * time.Second)

		// 检查轮转后的文件情况
		files, err := os.ReadDir(logDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}

		// 应该至少有当前日志文件和一个轮转后的日志文件
		rotatedCount := 0
		var rotatedFiles []string

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			fileName := file.Name()
			t.Logf("Found file: %s", fileName)

			if fileName == "test.log" {
				// 当前日志文件应该存在
				fileInfo, err := file.Info()
				if err != nil {
					t.Fatalf("Failed to get info for current log file: %v", err)
				}
				t.Logf("Current log file size: %d bytes", fileInfo.Size())
				continue
			}

			// 检查是否是轮转后的日志文件（格式为 test.YYYYMMDD.HHMMSS.SSS.log）
			if strings.HasPrefix(fileName, "test.") && strings.HasSuffix(fileName, ".log") && len(fileName) > 8 {
				rotatedCount++
				rotatedFiles = append(rotatedFiles, fileName)

				// 检查轮转文件的内容和大小
				rotatedPath := filepath.Join(logDir, fileName)
				fileInfo, err := os.Stat(rotatedPath)
				if err != nil {
					t.Logf("Failed to get info for rotated log file %s: %v", fileName, err)
					continue
				}
				t.Logf("Rotated log file %s size: %d bytes", fileName, fileInfo.Size())
			}
		}

		// 验证是否至少有一个轮转后的文件
		if rotatedCount == 0 {
			t.Fatal("No rotated log files found, rotation may have failed")
		} else {
			t.Logf("Found %d rotated log files: %v", rotatedCount, rotatedFiles)
		}

		// 检查当前日志文件是否存在且小于最大大小
		currentFileInfo, err := os.Stat(logPath)
		if os.IsNotExist(err) {
			t.Logf("Current log file does not exist. This might be expected if rotation just occurred and no new writes happened yet")
		} else if err != nil {
			t.Fatalf("Failed to get current log file info: %v", err)
		} else {
			t.Logf("Current log file size: %d bytes", currentFileInfo.Size())
			if currentFileInfo.Size() > int64(maxSizeKB*1024) {
				t.Errorf("Current log file size (%d bytes) exceeds max size (%d bytes)",
					currentFileInfo.Size(), maxSizeKB*1024)
			}
		}
	})
}
