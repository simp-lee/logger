package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	FormatText   = "text"
	FormatJSON   = "json"
	FormatCustom = "custom"

	DefaultTimeFormat    = "2006/01/02 15:04:05"
	DefaultMaxSizeMB     = 10
	DefaultRetentionDays = 7
	DefaultFormatter     = "{time} {level} {message} {file} {attrs}"
)

type Config struct {
	// Base configuration
	Level      slog.Level
	AddSource  bool
	TimeFormat string
	TimeZone   *time.Location
	Format     string // "json", "text", 或 "custom"

	// Console configuration
	Console   bool
	Color     bool
	Formatter string

	// File configuration
	File          bool
	FilePath      string
	MaxSizeMB     int
	RetentionDays int

	// ReplaceAttr is a function that can be used to replace attributes in log messages
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

func DefaultConfig() *Config {
	return &Config{
		Level:         slog.LevelInfo,
		AddSource:     false,
		TimeFormat:    DefaultTimeFormat,
		TimeZone:      time.Local,
		Format:        FormatCustom,
		Console:       true,
		Color:         true,
		Formatter:     DefaultFormatter,
		File:          false,
		MaxSizeMB:     DefaultMaxSizeMB,
		RetentionDays: DefaultRetentionDays,
		ReplaceAttr:   nil,
	}
}

// Option is a function that modifies a Config
type Option func(*Config)

func WithLevel(level slog.Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

func WithAddSource(addSource bool) Option {
	return func(c *Config) {
		c.AddSource = addSource
	}
}

func WithTimeFormat(timeFormat string) Option {
	return func(c *Config) {
		c.TimeFormat = timeFormat
	}
}

func WithTimeZone(timeZone *time.Location) Option {
	return func(c *Config) {
		c.TimeZone = timeZone
	}
}

func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

func WithConsole(console bool) Option {
	return func(c *Config) {
		c.Console = console
	}
}

func WithColor(enabled bool) Option {
	return func(c *Config) {
		c.Color = enabled
	}
}

// WithFormatter sets the formatter for logging, and automatically sets the format to FormatCustom
// The formatter string can contain the following placeholders:
// - {time}: The timestamp of the log message
// - {level}: The log level of the message
// - {message}: The log message
// - {file}: The source file where the log message was generated
// - {attrs}: Any additional attributes associated with the log message
// For example: "{time} [{level}] {file} {message} {attrs}"
func WithFormatter(formatter string) Option {
	return func(c *Config) {
		c.Formatter = formatter
	}
}

func WithFile(enabled bool) Option {
	return func(c *Config) {
		c.File = enabled
	}
}

func WithFilePath(path string) Option {
	return func(c *Config) {
		c.FilePath = path
		c.File = true
	}
}

// WithMaxSizeMB sets the maximum size of the log file in megabytes
func WithMaxSizeMB(maxSizeMB int) Option {
	return func(c *Config) {
		c.MaxSizeMB = maxSizeMB
	}
}

// WithRetentionDays sets the number of days to retain log files
func WithRetentionDays(retentionDays int) Option {
	return func(c *Config) {
		c.RetentionDays = retentionDays
	}
}

// WithReplaceAttr sets a function that can be used to replace attributes in log messages
func WithReplaceAttr(replaceAttr func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(c *Config) {
		c.ReplaceAttr = replaceAttr
	}
}

func validateConfig(cfg *Config) error {
	// Validate level
	if cfg.Level < slog.LevelDebug-4 || cfg.Level > slog.LevelError+4 {
		return fmt.Errorf("invalid log level: %v (should be within reasonable range)", cfg.Level)
	}

	// Validate time format
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = DefaultTimeFormat
	}

	// Validate time zone
	if cfg.TimeZone == nil {
		cfg.TimeZone = time.Local
	}

	// Validate format
	validFormats := map[string]bool{FormatText: true, FormatJSON: true, FormatCustom: true}
	if _, ok := validFormats[cfg.Format]; !ok {
		return fmt.Errorf("unsupported format: %s (must be one of: text, json, custom)", cfg.Format)
	}

	// Validate file configuration
	if cfg.File {
		if cfg.FilePath == "" {
			return fmt.Errorf("file logging enabled but FilePath is empty")
		}

		// Create the log directory if it doesn't exist
		dir := filepath.Dir(cfg.FilePath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("unable to create log directory %s: %w", dir, err)
			}
		} else if err != nil {
			return fmt.Errorf("error checking log directory %s: %w", dir, err)
		}

		if cfg.MaxSizeMB <= 0 {
			cfg.MaxSizeMB = DefaultMaxSizeMB
		}

		if cfg.RetentionDays <= 0 {
			cfg.RetentionDays = DefaultRetentionDays
		}
	}

	// Make sure at least one logging destination is enabled
	if !cfg.Console && !cfg.File {
		return fmt.Errorf("neither console nor file logging is enabled")
	}

	return nil
}
