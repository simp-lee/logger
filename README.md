# Go Logger

A flexible, lightweight logger for Go built on the standard `log/slog` package, supporting multiple outputs, colored output, customizable formatting, and log rotation.

## Features

- Multiple output destinations (console, file)
- TEXT, JSON, and custom format support
- Colored console output (automatically disabled in file output)
- Automatic log file rotation and retention
- Structured logging with context support
- Independent configuration for console and file outputs
- Easy integration with Go's standard `log/slog` package

## Installation

```bash
go get github.com/simp-lee/logger
```

## Quick Start

```go
package main

import "github.com/simp-lee/logger"

func main() {
    log, err := logger.New()
    if err != nil {
        panic(err)
    }
    defer log.Close() // Important: always close to clean up resources

    log.Info("Hello, world!")
    log.Info("User login", "userId", 123, "username", "johndoe")
}
```

## Configuration

```go
log, err := logger.New(
    // Global options
    logger.WithLevel(slog.LevelDebug),
    logger.WithAddSource(true),
    logger.WithTimeFormat("2006-01-02 15:04:05.000"),
    
    // Console-specific options
    logger.WithConsole(true),
    logger.WithConsoleColor(true),
    logger.WithConsoleFormat(logger.FormatCustom),
    logger.WithConsoleFormatter("{time} [{level}] {message} {file} {attrs}"),
    
    // File-specific options
    logger.WithFile(true),
    logger.WithFilePath("/var/log/myapp.log"),
    logger.WithFileFormat(logger.FormatJSON),  // File uses JSON format
    logger.WithMaxSizeMB(100),
    logger.WithRetentionDays(30),
)
if err != nil {
    panic(err)
}
defer log.Close() // Important: clean up resources when done
```

## Configuration Options

### Global Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithLevel` | Set the minimum logging level | `slog.LevelInfo` |
| `WithAddSource` | Include source file information | `false` |
| `WithTimeFormat` | Format for timestamp | `"2006/01/02 15:04:05"` |
| `WithTimeZone` | Timezone for log timestamps | `time.Local` |
| `WithReplaceAttr` | Custom attribute transformation function | `nil` |

### Console Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithConsole` | Enable console logging | `true` |
| `WithConsoleColor` | Enable colored output in console | `true` |
| `WithConsoleFormat` | Console log format (`FormatText`, `FormatJSON`, or `FormatCustom`) | `FormatCustom` |
| `WithConsoleFormatter` | Custom format template for console | `"{time} {level} {message} {file} {attrs}"` |

### File Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithFile` | Enable file logging | `false` |
| `WithFilePath` | Path to the log file | `""` |
| `WithFileFormat` | File log format (`FormatText`, `FormatJSON`, or `FormatCustom`) | `FormatCustom` |
| `WithFileFormatter` | Custom format template for file | `"{time} {level} {message} {file} {attrs}"` |
| `WithMaxSizeMB` | Maximum size of log file before rotation (0 to disable rotation) | `10` |
| `WithRetentionDays` | Days to retain rotated log files | `7` |

### Compatibility Options

These options set both console and file configurations at once:

| Option | Description | Affects |
| ------ | ----------- | ------- |
| `WithFormat` | Set log format for both console and file | Console & File |
| `WithFormatter` | Set formatter for both console and file | Console & File |

## Custom Formatting

```go
logger.WithConsoleFormatter("{time} | {level} | {message} | {attrs}")
```

**Placeholders**:
- `{time}`: Timestamp: `2006-01-02 15:04:05`, `time.RFC3339`, `time.Kitchen`, etc.
- `{level}`: Log level: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `{message}`: Log message
- `{file}`: Source file information: `file.go:123`
- `{attrs}`: Additional fields

## Color Scheme

When color output is enabled, the logger applies the following color scheme:
- `DEBUG`: Bright Cyan
- `INFO`: Bright Green
- `WARN`: Bright Yellow
- `ERROR`: Bright Red

Error messages and error attributes are highlighted in red to make them easily identifiable. Source file information is displayed in faint text for better readability.

**Note**: Color codes are only applied to console output. File logging automatically disables color codes to ensure log files remain clean and readable.

## Different Formats for Console and File

You can configure different output formats for console and file logging:

```go
log, err := logger.New(
    // Console uses a custom format with color
    logger.WithConsoleFormat(logger.FormatCustom),
    logger.WithConsoleFormatter("{time} | {level} | {message}"),
    logger.WithConsoleColor(true),
    
    // File uses JSON format
    logger.WithFile(true),
    logger.WithFilePath("./logs/myapp.log"),
    logger.WithFileFormat(logger.FormatJSON),
)
if err != nil {
    panic(err)
}
defer log.Close()
```

## Log Rotation

File logs are automatically rotated when they reach the configured maximum size:

```go
log, err := logger.New(
    logger.WithFile(true),
    logger.WithFilePath("./logs/myapp.log"),
    logger.WithMaxSizeMB(10),       // Rotate at 10MB
    logger.WithRetentionDays(7),    // Keep logs for 7 days
)
if err != nil {
    panic(err)
}
defer log.Close() // Important: ensures proper cleanup of rotation goroutines and timers
```

When a log file reaches the specified size limit, it is automatically renamed with a timestamp suffix in the format `{original_name}.YYYYMMDD.HHMMSS.SSS` and a new empty log file is created. Rotated log files older than the retention period are automatically cleaned up once per day.

**Note**: Setting `WithMaxSizeMB(0)` completely disables file rotation. The log file will continue to grow without any size limits. Negative values are reset to the default value (10MB).

## Multiple Destinations

The logger can write to both console and file simultaneously:

```go
log, err := logger.New(
    logger.WithConsole(true),
    logger.WithFile(true),
    logger.WithFilePath("./logs/myapp.log"),
)
if err != nil {
    panic(err)
}
defer log.Close()
```

When multiple destinations are configured, log messages are written to all enabled destinations in parallel to maximize throughput. Each handler independently evaluates whether to process the log record based on its level configuration.

## Standard Library Integration

```go
log, err := logger.New()
if err != nil {
    panic(err)
}
defer log.Close()

log.SetDefault()

// Now standard library calls use your configured logger
slog.Info("This uses your custom logger")
slog.Debug("Debug message via standard library")
slog.Error("Error message via standard library", "error", "some error")

// Standard library grouping and attributes also work
authLogger := slog.With("component", "auth")
authLogger.Info("Login attempt", "success", true)

dbLogger := slog.Default().WithGroup("database")
dbLogger.Info("Connection established", "host", "localhost")
```

The `SetDefault()` method calls `slog.SetDefault()` with your configured logger, making all standard library logging operations use your custom configuration.

## Advanced Example

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"
    
    "github.com/simp-lee/logger"
)

func main() {
    // Create a logger with custom configuration
    log, err := logger.New(
        // Global options
        logger.WithLevel(slog.LevelDebug),
        logger.WithAddSource(true),
        logger.WithTimeFormat("2006-01-02 15:04:05.000"),
        
        // Console options
        logger.WithConsole(true),
        logger.WithConsoleColor(true),
        logger.WithConsoleFormatter("{time} [{level}] {file} {message} {attrs}"),
        
        // File options
        logger.WithFile(true),
        logger.WithFilePath("./logs/application.log"),
        logger.WithFileFormat(logger.FormatJSON),
        logger.WithMaxSizeMB(50),
        logger.WithRetentionDays(30),
        
        // Custom attribute handling
        logger.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == "user_id" && len(groups) == 0 {
                return slog.String("userId", a.Value.String())
            }
            return a
        }),
    )
    if err != nil {
        panic(err)
    }
    defer log.Close() // Important: clean up background goroutines and timers
    
    log.SetDefault() // Make this the default logger
    
    // Basic logging
    log.Debug("Debug message")
    log.Info("Info message")
    log.Warn("Warning message")
    log.Error("Error occurred", "error", "sample error")
    
    // With additional context
    ctx := context.Background()
    log.InfoContext(ctx, "Message with context")
    
    // With structured fields
    log.Info("User action", 
        "user_id", 12345,
        "action", "login",
        "timestamp", time.Now().Unix(),
    )
    
    // Using groups for organization
    userLogger := log.WithGroup("user")
    userLogger.Info("User profile updated", "id", 12345)
    
    // Standard library integration
    slog.Debug("This uses the configured logger too")
}
```

## Performance

Thread-safe concurrent logging with benchmark test data:

### Performance Benchmarks

| Test Scenario | Performance | Memory | Notes |
|---------------|-------------|---------|-------|
| **Memory Output** | ~1,549 ns/op | 1,273 B/op (18 allocs) | Baseline |
| **File Output** | ~2,752,644 ns/op | 1,150 B/op (8 allocs) | I/O bound |
| **Text Format** | ~1,627 ns/op | 913 B/op (18 allocs) | Human readable |
| **JSON Format** | ~1,570 ns/op | 913 B/op (18 allocs) | Structured data |
| **Custom Format** | ~1,657 ns/op | 913 B/op (18 allocs) | Flexible |
| **With Color** | ~4,520 ns/op | 2,218 B/op (28 allocs) | Color processing |
| **Without Color** | ~4,310 ns/op | 1,722 B/op (24 allocs) | ~4.6% faster |
| **No Rotation** | ~3,230,165 ns/op | 1,235 B/op (8 allocs) | Rotation disabled |
| **With Rotation** | ~3,390,838 ns/op | 1,243 B/op (8 allocs) | ~5% overhead vs no rotation |
| **Concurrent Logging** | ~1,630 ns/op | 873 B/op (15 allocs) | Multi-threaded |
| **Multi-Handler** | ~5,044 ns/op | 2,371 B/op (40 allocs) | Multiple outputs |

### Concurrent Tests
| Scenario | Goroutines | Messages | Throughput | Status |
|----------|------------|----------|-----------|--------|
| Basic Concurrent | 100 | 1,000 | ~60,000 msg/sec | ✅ |
| Multi-Handler | 50 | 250 | ~25,000 msg/sec | ✅ |
| File Rotation | 20 | 1,000 | ~1,200 msg/sec | ✅ |
| High-Load Stress | 200 | 20,000 | ~310K-430K msg/sec | ✅ |

## License

MIT License