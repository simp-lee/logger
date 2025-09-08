package logger

import (
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
	closed       bool // Add a flag to track if the writer is closed
}

// newRotatingWriter creates a new rotatingWriter instance.
func newRotatingWriter(cfg *rotatingConfig) (*rotatingWriter, error) {
	w := &rotatingWriter{
		config:       cfg,
		rotateSignal: make(chan struct{}, 1),
	}

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

	filePath := filepath.Join(w.config.directory, w.config.fileName)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Get the current file info
	info, err := file.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	// Write the log message
	n, err = file.Write(p)
	if err != nil {
		return n, fmt.Errorf("failed to write to log file: %w", err)
	}

	// Update the size of the file
	size := info.Size() + int64(n)
	// Only check for rotation if MaxSizeMB > 0 (0 means rotation is disabled)
	// Also check if writer is still open before sending to channel
	if w.config.maxSizeMB > 0 && size > int64(w.config.maxSizeMB)*1024*1024 && !w.closed {
		select {
		case w.rotateSignal <- struct{}{}:
			// Signal sent successfully
		default:
			// Channel is full, rotation is already scheduled
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

	return nil
}

func (w *rotatingWriter) cleanOldLogs(ctx context.Context) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Calculate the cutoff time for log files
	cutoffTime := time.Now().AddDate(0, 0, -w.config.retentionDays)

	// Read the log directory
	entries, err := os.ReadDir(w.config.directory)
	if err != nil {
		slog.Warn("Error reading directory",
			slog.String("directory", w.config.directory),
			slog.Any("error", err),
		)
		return
	}

	var removed, retained, skipped int
	for _, entry := range entries {
		select {
		case <-ctx.Done():
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
				strings.TrimSuffix(w.config.fileName, filepath.Ext(w.config.fileName)),
			) &&
				entry.Name() != w.config.fileName

			if !isRotatedLog {
				skipped++
				continue
			}

			info, err := entry.Info()
			if err != nil {
				slog.Warn("Error getting file info",
					"file", entry.Name(),
					slog.Any("error", err),
				)
				continue
			}

			// Remove files older than the cutoff time
			if info.ModTime().Before(cutoffTime) {
				if err := os.Remove(filepath.Join(w.config.directory, entry.Name())); err != nil {
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
	return nil
}
