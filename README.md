# Go Logger

Lightweight, flexible logging library built on Go's `log/slog` with multi-output support, structured fields, custom templates, colors, and automatic rotation.

## Features

- **Multi-output**: Console & file with independent configuration
- **Flexible formats**: Text, JSON, and custom template support
- **Smart coloring**: ANSI colors for console (auto-disabled for files)
- **Auto-rotation**: Size-based rotation with configurable retention
- **Structured logging**: Full `slog` API with groups and attributes
- **Attribute transformation**: Custom processing via `WithReplaceAttr`
- **Drop-in replacement**: Embeds `*slog.Logger` + easy `SetDefault()`
- **High performance**: Lock-free formatting + efficient I/O

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
    defer log.Close()

    log.Info("Hello, world!")
    log.Info("User login", "userId", 123, "username", "johndoe")
}
```

## Quick Configuration Reference

All options are functional options passed to `logger.New(...)`. Only configure what you need; unspecified fields fall back to sane defaults.

## Configuration Options

### Global Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithLevel` | Set the minimum logging level | `slog.LevelInfo` |
| `WithAddSource` | Include source file information | `false` |
| `WithTimeFormat` | Format for timestamp | `"2006/01/02 15:04:05"` |
| `WithTimeZone` | Time zone for timestamps | `time.Local` |
| `WithReplaceAttr` | Custom attribute transformation function | `nil` |

### Console Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithConsole` | Enable console logging | `true` |
| `WithConsoleColor` | Enable colored output in console | `true` |
| `WithConsoleFormat` | Console log format (`FormatText`, `FormatJSON`, `FormatCustom`) | `FormatCustom` |
| `WithConsoleFormatter` | Custom format template for console | `"{time} {level} {message} {file} {attrs}"` |

### File Options

| Option | Description | Default |
| ------ | ----------- | ------- |
| `WithFile` | Enable file logging | `false` |
| `WithFilePath` | Path to the log file (enables file logging automatically) | `""` |
| `WithFileFormat` | File log format (`FormatText`, `FormatJSON`, `FormatCustom`) | `FormatCustom` |
| `WithFileFormatter` | Custom format template for file | `"{time} {level} {message} {file} {attrs}"` |
| `WithMaxSizeMB` | Maximum size of log file before rotation (0 to disable rotation) | `10` |
| `WithRetentionDays` | Days to retain rotated log files (<=0 resets to default) | `7` |

### Compatibility Options

These options set both console and file configurations at once:

| Option | Description | Affects |
| ------ | ----------- | ------- |
| `WithFormat` | Set log format for both console and file | Console & File |
| `WithFormatter` | Set formatter for both console and file | Console & File |

## Custom Formatting

Templates (`FormatCustom`) accept these placeholders:

| Placeholder | Meaning |
|-------------|---------|
| `{time}` | Timestamp (formatted via `WithTimeFormat` + `WithTimeZone`) |
| `{level}` | Log level string (`DEBUG`, `INFO`, `WARN`, `ERROR`) |
| `{message}` | Log message text |
| `{file}` | `filename:function:line` (only if `WithAddSource(true)`) |
| `{attrs}` | User attributes (key=value ...) |

Example:
```go
log, err := logger.New(
    logger.WithConsoleFormatter("{time} | {level} | {message} | {attrs}"),
)
if err != nil {
    panic(err)
}
defer log.Close()
```
If a placeholder produces empty content (e.g. `{file}` without source), surrounding extra spaces are minimized automatically.

## Color Output

Console coloring (ANSI) can be toggled with `WithConsoleColor(true/false)`. File output never includes color. Levels map to Bright Cyan / Green / Yellow / Red; error messages & `error` attribute keys are emphasized.

## Mixed Formats (Console vs File)

Formats are independent. Common pattern: human-readable console + structured file:
```go
log, err := logger.New(
    logger.WithConsoleFormat(logger.FormatCustom),
    logger.WithConsoleFormatter("{time} {level} {message}"),
    logger.WithConsoleColor(true),
    logger.WithFilePath("./logs/app.log"),
    logger.WithFileFormat(logger.FormatJSON),
)
if err != nil {
    panic(err)
}
defer log.Close()
```

## File Rotation & Retention

- Trigger: size > `WithMaxSizeMB(N)` MB. Set `0` to disable rotation. Negative values reset to default (10MB).
- Naming pattern: `basename.YYYYMMDD.HHMMSS.mmm.ext` (adds `.counter` if collision).
- Retention: Files older than `WithRetentionDays(D)` days are purged daily; `<=0` resets to default (7 days).

Example:
```go
log, err := logger.New(
    logger.WithFilePath("./logs/app.log"),
    logger.WithMaxSizeMB(50),
    logger.WithRetentionDays(30),
)
if err != nil {
    panic(err)
}
defer log.Close()
```

## Multiple Outputs

Enable console + file together. `WithFilePath` implies `WithFile(true)`. At least one destination must remain enabled; disabling both returns an error.

## Attribute Transformation (`WithReplaceAttr`)

Intercept & edit/remove attributes (including built-ins: time, level, message, source, and user attrs). Return an empty `slog.Attr{}` to drop an attribute.
```go
log, err := logger.New(
    logger.WithReplaceAttr(func(groups []string, a slog.Attr) slog.Attr {
        if a.Key == "user_id" {
            return slog.String("userId", a.Value.String())
        }
        return a
    }),
)
if err != nil {
    panic(err)
}
defer log.Close()
```

## Standard Library Integration

Because `Logger` embeds `*slog.Logger`, you get the full `slog` API. Call `SetDefault()` to route global `slog.*` calls:
```go
log, err := logger.New(logger.WithAddSource(true))
if err != nil {
    panic(err)
}
defer log.Close()

log.SetDefault()
slog.Info("uses custom logger", "module", "auth")
```

## Complete Example
```go
package main

import (
    "log/slog"
    "github.com/simp-lee/logger"
)

func main() {
    log, err := logger.New(
        logger.WithLevel(slog.LevelDebug),
        logger.WithAddSource(true),
        logger.WithConsoleFormatter("{time} [{level}] {message} {attrs}"),
        logger.WithFilePath("./logs/app.log"),
        logger.WithFileFormat(logger.FormatJSON),
    )
    if err != nil {
        panic(err)
    }
    defer log.Close()
    
    log.Info("startup", "version", "1.0.0")
    userLogger := log.WithGroup("user")
    userLogger.Info("login", "id", 42)
}
```

## Performance

Thread-safe with optimized performance (Windows amd64, i5-13500H).

### Performance Benchmarks

| Test Scenario | Performance (ns/op) | Memory (B/op) | Allocations | Notes |
|---------------|---------------------|---------------|-------------|-------|
| Memory Output | ~367 | 1,055 | 17 | Baseline performance |
| File Output | ~8,730 | 24 | 2 | Optimized I/O with buffering |
| Text Format | ~367 | 738 | 17 | Human readable |
| JSON Format | ~359 | 738 | 17 | Structured data |
| Custom Format | ~390 | 737 | 17 | Flexible formatting |
| With Color | ~871 | 1,234 | 22 | Enhanced readability |
| Without Color | ~791 | 889 | 18 | ~9% faster than colored |
| No Rotation | ~8,996 | 32 | 2 | File rotation disabled |
| With Rotation | ~8,908 | 32 | 2 | Minimal rotation overhead (~1%) |
| Concurrent Logging | ~373 | 698 | 14 | Multi-threaded safe |
| Multi-Handler | ~687 | 1,454 | 31 | Multiple output targets |

### Concurrent Tests
| Scenario | Goroutines | Messages | Throughput (msg/sec) |
|----------|------------|----------|---------------------|
| Basic Concurrent | 100 | 100,000 | ~1,799,474 |
| Multi-Handler | 50 | 5,000 | ~242,502 |
| File Rotation | 20 | 10,000 | ~108,780 |
| High-Load Stress | 200 | 20,000 | ~701,498 |

**Key insights**: Concurrent logging achieves ~2.7M msg/sec theoretical peak. Production workloads typically see 500K-1.8M msg/sec depending on I/O patterns and template complexity.

## License

MIT License