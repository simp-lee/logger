package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/simp-lee/logger"
)

func main() {
	os.MkdirAll("./logs", 0755)

	simpleExample()
	fileExample()
	resourceExample()
	rotationExample()
}

// Basic example
func simpleExample() {
	printTitle("Basic Example")

	log, err := logger.New()
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("This is an info log")
	log.Warn("This is a warning log")
	log.Error("This is an error log")
}

// File logging example
func fileExample() {
	printTitle("File Logging Example")

	log, err := logger.New(
		logger.WithFilePath("./logs/app.log"),
		logger.WithFileFormat(logger.FormatJSON),
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("This log is written to file", "user", "john", "action", "login")
	log.Error("Error occurred", "error", "connection timeout")

	// Read and display file content
	content, _ := os.ReadFile("./logs/app.log")
	fmt.Printf("File content:\n%s\n", content)
}

// Resource management example
func resourceExample() {
	printTitle("Resource Management Example")

	log, err := logger.New(
		logger.WithFilePath("./logs/resource.log"),
		logger.WithMaxSizeMB(1),
		logger.WithRetentionDays(7),
	)
	if err != nil {
		panic(err)
	}

	// This ensures proper cleanup
	defer func() {
		if err := log.Close(); err != nil {
			fmt.Printf("Error closing logger: %v\n", err)
		} else {
			fmt.Println("Logger resources cleaned up successfully")
		}
	}()

	log.Info("Background goroutines and timers will be properly cleaned up")
	log.Warn("No resource leaks!")
}

// File rotation example
func rotationExample() {
	printTitle("File Rotation Example")

	log, err := logger.New(
		logger.WithFilePath("./logs/rotation.log"),
		logger.WithMaxSizeMB(1), // Small size to trigger rotation
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	// Write enough data to trigger rotation
	for i := 0; i < 1000; i++ {
		log.Info("Log rotation test", "iteration", i, "data", strings.Repeat("x", 1000))
	}

	fmt.Println("Check ./logs directory for rotated files")

	// List rotated files
	entries, _ := os.ReadDir("logs")
	fmt.Println("Files in logs directory:")
	for _, entry := range entries {
		if !entry.IsDir() && strings.Contains(entry.Name(), "rotation") {
			info, _ := entry.Info()
			fmt.Printf(" - %s (Size: %.2f KB)\n", entry.Name(), float64(info.Size())/1024)
		}
	}
}

func printTitle(title string) {
	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("  %s\n", title)
	fmt.Printf("%s\n", strings.Repeat("=", 50))
}

// File logging example
func fileLogExample() {
	printTitle("File Logging Example")

	logPath := filepath.Join("logs", "app.log")
	log, err := logger.New(
		logger.WithConsole(false),
		logger.WithFile(true),
		logger.WithFilePath(logPath),
		logger.WithFileFormat(logger.FormatText),
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("This log is written to file only, not displayed in console")

	// Read and display file content to demonstrate
	content, err := os.ReadFile(logPath)
	if err == nil {
		println("File content:")
		println(" " + string(content))
	}
}

// Multi-destination output example
func multiDestinationExample() {
	printTitle("Multi-Destination Output Example")

	logPath := filepath.Join("logs", "multi.log")
	log, err := logger.New(
		logger.WithConsole(true),
		logger.WithConsoleColor(true),
		logger.WithFile(true),
		logger.WithFilePath(logPath),
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("This log is written to both console and file")
	log.Error("This is an error log, displayed in red in the console")

	// Read and display file content to demonstrate
	content, err := os.ReadFile(logPath)
	if err == nil {
		println("File content:")
		println(" " + string(content))
	}
}

// Color control example
func colorControlExample() {
	printTitle("Color Control Example")

	// With color
	colorLog, _ := logger.New(
		logger.WithConsoleColor(true),
	)
	defer colorLog.Close()
	colorLog.Info("This is a colored info log")
	colorLog.Error("This is a colored error log")

	// Without color
	noColorLog, _ := logger.New(
		logger.WithConsoleColor(false),
	)
	defer noColorLog.Close()
	noColorLog.Info("This is a info log without color")
	noColorLog.Error("This is an error log without color")
}

// Structured logging example
func structuredLoggingExample() {
	printTitle("Structured Logging Example")

	log, _ := logger.New()
	defer log.Close()

	// Log with attributes
	log.Info("User login",
		"userId", 12345,
		"username", "John Smith",
		"loginTime", time.Now().Format(time.RFC3339))

	// Log with error
	err := os.ErrNotExist
	log.Error("Operation failed", "error", err, "operation", "read file")

	// Using With method to add context
	userLog := log.With("module", "user management", "traceId", "abc-123")
	userLog.Info("Update user information")
	userLog.Error("User verification failed")
}

// Custom format example
func customFormatterExample() {
	printTitle("Custom Format Example")

	// Simple format
	simpleLog, _ := logger.New(
		logger.WithConsoleFormatter("{time} > {message}"),
	)
	defer simpleLog.Close()
	simpleLog.Info("This is the simplest format")

	// Verbose format
	verboseLog, _ := logger.New(
		logger.WithConsoleFormatter("{time} | [{level}] | {file} | {message} | {attrs}"),
		logger.WithAddSource(true),
	)
	defer verboseLog.Close()
	verboseLog.Info("This is a detailed format", "key1", "value1", "key2", "value2")
}

// Group example
func groupExample() {
	printTitle("Group Example")

	log, _ := logger.New()
	defer log.Close()

	// Create group
	dbLog := log.WithGroup("Database")
	dbLog.Info("Connect to database")
	dbLog.Error("Query failed", "error", "connection timeout")

	// Nested group
	mysqlLog := dbLog.WithGroup("MySQL")
	mysqlLog.Info("Execute query", "query", "SELECT * FROM users")
}

// Different console and file formats example
func differentFormatsExample() {
	printTitle("Different Console and File Formats Example")

	logPath := filepath.Join("logs", "different_formats.log")
	log, err := logger.New(
		// Console uses custom format
		logger.WithConsoleFormat(logger.FormatCustom),
		logger.WithConsoleFormatter("{time} | {level} | {message}"),
		logger.WithConsoleColor(true),

		// File uses JSON format
		logger.WithFile(true),
		logger.WithFilePath(logPath),
		logger.WithFileFormat(logger.FormatJSON),
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	log.Info("Console shows custom format, file stores as JSON", "key1", "value1")

	// Read and display file content
	content, err := os.ReadFile(logPath)
	if err == nil {
		println("File content (JSON):")
		println(" " + string(content))
	}
}

// Standard library integration example
func standardLibraryExample() {
	printTitle("Standard Library Integration Example")

	// Create a logger with specific configuration
	log, err := logger.New(
		logger.WithLevel(slog.LevelDebug),
		logger.WithConsoleFormat(logger.FormatCustom),
		logger.WithConsoleFormatter("{time} | {level} | {message} | {attrs}"),
		logger.WithConsoleColor(true),
	)
	if err != nil {
		panic(err)
	}
	defer log.Close()

	// Set as default logger for the entire application
	log.SetDefault()

	// Now standard library calls use your configured logger
	println("Using slog package directly with our custom logger:")

	// Basic logging with standard slog package
	slog.Debug("This is a debug message via slog package")
	slog.Info("This is an info message via slog package")
	slog.Warn("This is a warning message via slog package")
	slog.Error("This is an error message via slog package")

	// Structured logging with standard slog package
	slog.Info("User authenticated via slog package",
		"user", "admin",
		"ip", "192.168.1.1",
		"timestamp", time.Now().Unix(),
	)

	// Using With methods
	authLog := slog.With("component", "auth")
	authLog.Info("Login attempt", "success", true, "method", "password")

	// Nested group
	dbLog := slog.Default().WithGroup("database")
	dbLog.Info("Connection established", "host", "localhost", "port", 5432)

	println("\nNote how all the above logs use our custom format and colors!")
	println("This is useful when third-party libraries use the standard logger.")
}

// Resource management example - proper cleanup of logger resources
func resourceManagementExample() {
	printTitle("Resource Management Example - Proper Cleanup")

	// Create logger with proper resource management
	log, err := logger.New(
		logger.WithFilePath("./logs/resource_example.log"),
		logger.WithFileFormat(logger.FormatJSON),
		logger.WithMaxSizeMB(1),
		logger.WithRetentionDays(7),
	)
	if err != nil {
		panic(err)
	}

	// Use defer to ensure proper cleanup
	defer func() {
		if err := log.Close(); err != nil {
			println("Error closing logger:", err.Error())
		} else {
			println("Logger resources cleaned up successfully")
		}
	}()

	// Use the logger
	log.Info("This log will be written to file with proper resource management")
	log.Warn("Background goroutines and timers will be properly cleaned up")
	log.Error("No resource leaks!")

	println("Logger created and used. Resources will be cleaned up on function exit.")
}
