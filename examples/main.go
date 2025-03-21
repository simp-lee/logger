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
	// Ensure log directory exists
	logDir := "./logs"
	os.MkdirAll(logDir, 0755)

	// Basic example - Default configuration
	simpleExample()

	// Console format example
	consoleFormatExample()

	// File logging example
	fileLogExample()

	// Multi-destination output example
	multiDestinationExample()

	// Color control example
	colorControlExample()

	// Structured logging example
	structuredLoggingExample()

	// Custom format example
	customFormatterExample()

	// Group example
	groupExample()

	// Different console and file formats example
	differentFormatsExample()

	// Standard library integration example
	standardLibraryExample()

	// File rotation example (write large amount of logs to trigger rotation)
	rotationExample()
}

// Basic example - using default configuration
func simpleExample() {
	printTitle("Basic Example - Default Configuration")

	// Create default handler
	handler, err := logger.NewHandler()
	if err != nil {
		panic(err)
	}

	// Create logger
	log := logger.New(handler)

	// Log different levels
	log.Debug("This is a debug log, not displayed by default")
	log.Info("This is an info log")
	log.Warn("This is a warning log")
	log.Error("This is an error log")
}

// Console format example
func consoleFormatExample() {
	printTitle("Console Format Example - Text, JSON, Custom")

	// Text format
	textHandler, _ := logger.NewHandler(
		logger.WithConsoleFormat(logger.FormatText),
	)
	textLog := logger.New(textHandler)
	textLog.Info("This is a TEXT format log")

	// JSON format
	jsonHandler, _ := logger.NewHandler(
		logger.WithConsoleFormat(logger.FormatJSON),
	)
	jsonLog := logger.New(jsonHandler)
	jsonLog.Info("This is a JSON format log")

	// Custom format
	customHandler, _ := logger.NewHandler(
		logger.WithConsoleFormatter("[{time}] {level}: {message}"),
	)
	customLog := logger.New(customHandler)
	customLog.Info("This is a custom format log")
}

// File logging example
func fileLogExample() {
	printTitle("File Logging Example")

	logPath := filepath.Join("logs", "app.log")
	handler, err := logger.NewHandler(
		logger.WithConsole(false),
		logger.WithFile(true),
		logger.WithFilePath(logPath),
		logger.WithFileFormat(logger.FormatText),
	)
	if err != nil {
		panic(err)
	}

	log := logger.New(handler)
	log.Info("This log is written to file only, not displayed in console")

	// Print file content
	content, _ := os.ReadFile(logPath)
	println("File content:\n", string(content))
}

// Multi-destination output example
func multiDestinationExample() {
	printTitle("Multi-Destination Output Example")

	logPath := filepath.Join("logs", "multi.log")
	handler, err := logger.NewHandler(
		logger.WithConsole(true),
		logger.WithConsoleColor(true),
		logger.WithFile(true),
		logger.WithFilePath(logPath),
	)
	if err != nil {
		panic(err)
	}

	log := logger.New(handler)
	log.Info("This log is written to both console and file")
	log.Error("This is an error log, displayed in red in the console")

	// Print file content
	content, _ := os.ReadFile(logPath)
	println("File content:\n", string(content))
}

// Color control example
func colorControlExample() {
	printTitle("Color Control Example")

	// Enable color
	colorHandler, _ := logger.NewHandler(
		logger.WithConsoleColor(true),
	)
	colorLog := logger.New(colorHandler)
	colorLog.Info("This is a colored info log")
	colorLog.Error("This is a colored error log")

	// Disable color
	noColorHandler, _ := logger.NewHandler(
		logger.WithConsoleColor(false),
	)
	noColorLog := logger.New(noColorHandler)
	noColorLog.Info("This is a info log without color")
	noColorLog.Error("This is an error log without color")
}

// Structured logging example
func structuredLoggingExample() {
	printTitle("Structured Logging Example")

	handler, _ := logger.NewHandler()
	log := logger.New(handler)

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
	simpleHandler, _ := logger.NewHandler(
		logger.WithConsoleFormatter("{time} > {message}"),
	)
	simpleLog := logger.New(simpleHandler)
	simpleLog.Info("This is the simplest format")

	// Verbose format
	verboseHandler, _ := logger.NewHandler(
		logger.WithConsoleFormatter("{time} | [{level}] | {file} | {message} | {attrs}"),
		logger.WithAddSource(true),
	)
	verboseLog := logger.New(verboseHandler)
	verboseLog.Info("This is a detailed format", "key1", "value1", "key2", "value2")
}

// Group example
func groupExample() {
	printTitle("Group Example")

	handler, _ := logger.NewHandler()
	log := logger.New(handler)

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
	handler, err := logger.NewHandler(
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

	log := logger.New(handler)
	log.Info("Console shows custom format, file stores as JSON", "key1", "value1")

	// Print file content
	content, _ := os.ReadFile(logPath)
	println("File content (JSON):\n", string(content))
}

// Standard library integration example
func standardLibraryExample() {
	printTitle("Standard Library Integration Example")

	// Create a handler with specific configuration
	handler, err := logger.NewHandler(
		logger.WithLevel(slog.LevelDebug),
		logger.WithConsoleColor(true),
		logger.WithConsoleFormatter("{time} | {level} | {message} | {attrs}"),
	)
	if err != nil {
		panic(err)
	}

	// Create logger and set it as the default for standard library
	log := logger.New(handler)
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
		"timestamp", time.Now().Unix())

	// Using WithGroup with standard slog
	authLogger := slog.With("component", "auth")
	authLogger.Info("Login attempt", "success", true, "method", "password")

	// Using WithGroup with standard slog
	dbLogger := slog.Default().WithGroup("database")
	dbLogger.Info("Connection established", "host", "localhost", "port", 5432)

	println("\nNote how all the above logs use our custom format and colors!")
	println("This is useful when third-party libraries use the standard logger.")
}

// File rotation example
func rotationExample() {
	printTitle("File Rotation Example")

	logPath := filepath.Join("logs", "rotation.log")
	handler, err := logger.NewHandler(
		logger.WithConsole(false),
		logger.WithFile(true),
		logger.WithFilePath(logPath),
		logger.WithMaxSizeMB(1), // Set to 1MB to trigger rotation
		logger.WithRetentionDays(7),
	)
	if err != nil {
		panic(err)
	}

	log := logger.New(handler)
	log.Info("Start writing large amount of logs to trigger file rotation...")

	// Write a large amount of logs to trigger rotation
	message := "This is a test log used to fill the file to trigger the rotation mechanism. " +
		"You can check the logs directory to see if multiple log files are generated."

	// Write about 20,000 logs
	for i := 0; i < 20000; i++ {
		log.Info(message, "index", i, "timestamp", time.Now().UnixNano())
	}

	println("Log writing completed, please check if there are rotation files in the logs directory")

	// List all log files
	entries, _ := os.ReadDir("logs")
	println("Files in logs directory:")
	for _, entry := range entries {
		if !entry.IsDir() && (entry.Name() == "rotation.log" ||
			strings.HasPrefix(entry.Name(), "rotation.")) {
			info, _ := entry.Info()
			println(" - " + entry.Name() +
				"  (Size: " + fmt.Sprintf("%.2f KB", float64(info.Size())/1024) + ")")
		}
	}
}

// Helper function: print title
func printTitle(title string) {
	println("\n" + strings.Repeat("=", 50))
	println("  " + title)
	println(strings.Repeat("=", 50))
}
