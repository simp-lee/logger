package logger

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// rotatingConfig defines parameters for log file rotation
type rotatingConfig struct {
	directory     string // Directory to store log files
	fileName      string // Base name of the log file
	maxSizeMB     int    // Maximum size in MB before rotation
	retentionDays int    // Number of days to keep log files
}

// rotatingWriter handles log file rotation and writing.
type rotatingWriter struct {
	config       *rotatingConfig
	mutex        sync.Mutex
	rotateSignal chan struct{}
	cleanupTimer *time.Timer
	closed       bool // flag to track if the writer is closed
	file         *os.File
	buf          *bufio.Writer
	currentSize  int64 // bytes written to current file (including buffered)
}

// newRotatingWriter creates a new rotatingWriter instance.
func newRotatingWriter(cfg *rotatingConfig) (*rotatingWriter, error) {
	w := &rotatingWriter{
		config:       cfg,
		rotateSignal: make(chan struct{}, 1),
	}
	// NOTE: we intentionally do NOT open the file here to avoid
	// keeping descriptors open for handlers that are constructed
	// but never used in tests (some tests create a handler and never write).
	// The file is opened lazily on first Write or after rotation.

	// Start the rotation monitor
	go w.rotateMonitor()

	// Set up the cleanup timer to run once a day
	w.cleanupTimer = time.AfterFunc(timeUntilNextDay(), func() {
		w.cleanOldLogs(context.Background())
		// Reschedule the cleanup every 24 hours
		w.cleanupTimer.Reset(time.Hour * 24)
	})

	return w, nil
}

// timeUntilNextDay returns the duration until the next day.
func timeUntilNextDay() time.Duration {
	now := time.Now()
	next := now.Add(24 * time.Hour)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	return next.Sub(now)
}

// rotateMonitor listens for rotation signals and performs log rotation.
func (w *rotatingWriter) rotateMonitor() {
	for range w.rotateSignal {
		if err := w.rotate(); err != nil {
			// Log the error, but continue operating
			slog.Warn("Error during log rotation", slog.Any("error", err))
		}
	}
}

// Write implements io.Writer interface for rotatingWriter.
func (w *rotatingWriter) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Check if the writer has been closed to avoid panic on closed channel
	if w.closed {
		return 0, fmt.Errorf("writer has been closed")
	}
	if w.file == nil || w.buf == nil { // should not happen, but be defensive
		if err := w.openCurrentFile(); err != nil {
			return 0, err
		}
	}

	n, err = w.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to write to buffer: %w", err)
	}
	// Flush immediately to satisfy tests that read the file right after Write.
	if err := w.buf.Flush(); err != nil {
		return n, fmt.Errorf("failed to flush buffer: %w", err)
	}
	w.currentSize += int64(n)

	// Rotation check (include buffered data)
	if w.config.maxSizeMB > 0 && w.currentSize > int64(w.config.maxSizeMB)*1024*1024 && !w.closed {
		select {
		case w.rotateSignal <- struct{}{}:
		default:
		}
	}
	return n, nil
}

// rotate performs log rotation by renaming the current log file.
func (w *rotatingWriter) rotate() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	oldPath := filepath.Join(w.config.directory, w.config.fileName)

	// Check if the file exists before rotating
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check log file: %w", err)
	}

	// Flush buffered data before rotation
	if w.buf != nil {
		_ = w.buf.Flush() // ignore flush error, we'll catch write/open errors later
	}
	if w.file != nil {
		// Close current file before renaming (required on Windows)
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("failed to close file before rotation: %w", err)
		}
		w.file = nil
		w.buf = nil
	}

	ext := filepath.Ext(w.config.fileName)
	timestamp := time.Now().Format("20060102.150405.000")

	// Generate a unique filename for the rotated log
	newPath := filepath.Join(w.config.directory, fmt.Sprintf("%s.%s%s",
		strings.TrimSuffix(w.config.fileName, ext),
		timestamp,
		ext))

	// Ensure the new path is unique by adding a counter if needed
	counter := 0
	for {
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			break
		}
		counter++
		newPath = filepath.Join(w.config.directory, fmt.Sprintf("%s.%s.%d%s",
			strings.TrimSuffix(w.config.fileName, ext),
			timestamp,
			counter,
			ext))
	}

	// Rename the current log file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Open a new current file
	if err := w.openCurrentFile(); err != nil {
		return fmt.Errorf("failed to open new log file after rotation: %w", err)
	}
	w.currentSize = 0
	return nil
}

func (w *rotatingWriter) cleanOldLogs(ctx context.Context) {
	w.mutex.Lock()
	cutoffTime := time.Now().AddDate(0, 0, -w.config.retentionDays)
	directory := w.config.directory
	fileName := w.config.fileName
	w.mutex.Unlock()

	// Read the log directory without holding the lock
	entries, err := os.ReadDir(directory)
	if err != nil {
		// Log the error without holding the lock to avoid deadlock
		slog.Warn("Error reading directory",
			slog.String("directory", directory),
			slog.Any("error", err),
		)
		return
	}

	var removed, retained, skipped int
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			// Log cleanup cancelled (without holding the lock)
			slog.Warn("Log cleanup cancelled",
				"removed", removed,
				"retained", retained,
				"skipped", skipped,
			)
			return
		default:
			if entry.IsDir() {
				skipped++
				continue
			}

			// Skip files that don't match the log file name
			isRotatedLog := strings.HasPrefix(
				entry.Name(),
				strings.TrimSuffix(fileName, filepath.Ext(fileName)),
			) &&
				entry.Name() != fileName

			if !isRotatedLog {
				skipped++
				continue
			}

			info, err := entry.Info()
			if err != nil {
				// Log the error without holding the lock to avoid deadlock
				slog.Warn("Error getting file info",
					"file", entry.Name(),
					slog.Any("error", err),
				)
				continue
			}

			// Remove files older than the cutoff time
			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
					// Log the error without holding the lock to avoid deadlock
					slog.Warn("Error removing old log file",
						"file", entry.Name(),
						slog.Any("error", err),
					)
				} else {
					removed++
				}
			} else {
				retained++
			}
		}
	}

	// Log the cleanup results without holding the lock
	slog.Info("Log cleanup completed",
		"removed", removed,
		"retained", retained,
		"skipped", skipped,
	)
}

// Close stops the cleanup timer and closes the rotatingWriter.
func (w *rotatingWriter) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Prevent multiple closes
	if w.closed {
		return nil
	}
	w.closed = true

	if w.cleanupTimer != nil {
		w.cleanupTimer.Stop()
	}
	close(w.rotateSignal)

	if w.buf != nil {
		_ = w.buf.Flush()
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
		w.file = nil
		w.buf = nil
	}
	return nil
}

// openCurrentFile opens or creates the current log file and prepares buffered writer.
func (w *rotatingWriter) openCurrentFile() error {
	if err := os.MkdirAll(w.config.directory, 0o755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	path := filepath.Join(w.config.directory, w.config.fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}
	w.file = f
	// 64KB buffer (reasonable default)
	w.buf = bufio.NewWriterSize(f, 64*1024)
	w.currentSize = info.Size()
	return nil
}
