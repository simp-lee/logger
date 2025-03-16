# Go Logger

A flexible, lightweight logger for Go built on the standard `log/slog` package, supporting multiple outputs, colored output, customizable formatting, and log rotation.

## Features

- Multiple output destinations (console, file)
- TEXT, JSON, and custom format support
- Colored console output
- Automatic log file rotation and retention
- Structured logging with context support

## Installation

```bash
go get github.com/simp-lee/logger
```

## Quick Start

```go
package main

import "github.com/simp-lee/logger"

func main() {
    handler, err := logger.NewHandler()
    if err != nil {
        panic(err)
    }

    log := logger.New(handler)
    log.Info("Hello, world!")
    log.Info("User login", "userId", 123, "username", "johndoe")
}
```

## Configuration

```go
handler, err := logger.NewHandler(
    logger.WithLevel(slog.LevelDebug),
    logger.WithAddSource(true),
    logger.WithTimeFormat("2006-01-02 15:04:05.000"),
    logger.WithConsole(true),
    logger.WithColor(true),
    logger.WithFormatter("{time} [{level}] {message} {file} {attrs}"),
    logger.WithFile(true),
    logger.WithFilePath("/var/log/myapp.log"),
    logger.WithMaxSizeMB(100),
    logger.WithRetentionDays(30),
)
```

## Configuration Options

### General Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithLevel` | Set the minimum logging level | `slog.LevelInfo` |
| `WithAddSource` | Include source file information | `false` |
| `WithTimeFormat` | Format for timestamp | `"2006/01/02 15:04:05"` |
| `WithTimeZone` | Timezone for log timestamps | `time.Local` |
| `WithFormat` | Log format (`"text"`, `"json"`, or `"custom"`) | `"custom"` |
| `WithReplaceAttr` | Custom attribute transformation function | `nil` |

### Console Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithConsole` | Enable console logging | `true` |
| `WithColor` | Enable colored output in console | `true` |
| `WithFormatter` | Custom log format template | `"{time} {level} {message} {file} {attrs}"` |

### File Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithFile` | Enable file logging | `false` |
| `WithFilePath` | Path to the log file | `""` |
| `WithMaxSizeMB` | Maximum size of log file before rotation | `10` |
| `WithRetentionDays` | Days to retain rotated log files | `7` |

## Custom Formatting

```go
logger.WithFormatter("{time} | {level} | {message} | {attrs}")
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

## Log Rotation

File logs are automatically rotated when they reach the configured maximum size:

```go
handler, err := logger.NewHandler(
    logger.WithFile(true),
    logger.WithFilePath("/var/log/myapp.log"),
    logger.WithMaxSizeMB(10),       // Rotate at 10MB
    logger.WithRetentionDays(7),    // Keep logs for 7 days
)
```

When a log file reaches the specified size limit, it is automatically renamed with a timestamp suffix in the format `{original_name}.YYYYMMDD.HHMMSS.SSS` and a new empty log file is created. Rotated log files older than the retention period are automatically cleaned up once per day.

## Multiple Destinations

The logger can write to both console and file simultaneously:

```go
handler, err := logger.NewHandler(
    logger.WithConsole(true),
    logger.WithFile(true),
    logger.WithFilePath("/var/log/myapp.log"),
)
```

When multiple destinations are configured, log messages are written to all enabled destinations in parallel to maximize throughput. Each handler independently evaluates whether to process the log record based on its level configuration.

## Standard Library Integration

```go
log := logger.New(handler)
log.SetDefault()

// Now standard library calls use your configured logger
slog.Info("This uses your custom logger")
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
    // Create a handler with custom configuration
    handler, err := logger.NewHandler(
        logger.WithLevel(slog.LevelDebug),
        logger.WithAddSource(true),
        logger.WithTimeFormat("2006-01-02 15:04:05.000"),
        logger.WithConsole(true),
        logger.WithColor(true),
        logger.WithFormatter("{time} [{level}] {file} {message} {attrs}"),
        logger.WithFile(true),
        logger.WithFilePath("./logs/application.log"),
        logger.WithMaxSizeMB(50),
        logger.WithRetentionDays(30),
        logger.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
            // Custom attribute handling
            if a.Key == "user_id" && len(groups) == 0 {
                return slog.String("userId", a.Value.String())
            }
            return a
        }),
    )
    if err != nil {
        panic(err)
    }
    
    log := logger.New(handler)
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

## License

MIT License