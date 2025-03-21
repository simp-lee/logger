package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type OutputFormat string

const (
	FormatText   OutputFormat = "text"
	FormatJSON   OutputFormat = "json"
	FormatCustom OutputFormat = "custom"

	DefaultTimeFormat    = "2006/01/02 15:04:05"
	DefaultMaxSizeMB     = 10
	DefaultRetentionDays = 7
	DefaultFormatter     = "{time} {level} {message} {file} {attrs}"
	DefaultFormat        = FormatText
)

type Config struct {
	// Base configuration
	Level      slog.Level
	AddSource  bool
	TimeFormat string
	TimeZone   *time.Location

	// Configurations for different log destinations
	Console ConsoleConfig
	File    FileConfig

	// ReplaceAttr is a function that can be used to replace attributes in log messages
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}

type ConsoleConfig struct {
	Enabled   bool         // Enable console logging
	Color     bool         // Enable colorized output
	Format    OutputFormat // text, json, custom
	Formatter string       // Custom formatter string, only used if Format is FormatCustom
}

type FileConfig struct {
	Enabled       bool
	Format        OutputFormat
	Formatter     string // Custom formatter string, only used if Format is FormatCustom
	Path          string // Path to the log file
	MaxSizeMB     int    // Maximum size of the log file in megabytes
	RetentionDays int    // Number of days to retain log files
}

func DefaultConfig() *Config {
	return &Config{
		Level:      slog.LevelInfo,
		AddSource:  false,
		TimeFormat: DefaultTimeFormat,
		TimeZone:   time.Local,

		Console: ConsoleConfig{
			Enabled:   true,
			Color:     true,
			Format:    FormatCustom,
			Formatter: DefaultFormatter,
		},

		File: FileConfig{
			Enabled:       false,
			Format:        FormatCustom,
			Formatter:     DefaultFormatter,
			Path:          "",
			MaxSizeMB:     DefaultMaxSizeMB,
			RetentionDays: DefaultRetentionDays,
		},

		ReplaceAttr: nil,
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

// WithReplaceAttr sets a function that can be used to replace attributes in log messages
func WithReplaceAttr(replaceAttr func(groups []string, a slog.Attr) slog.Attr) Option {
	return func(c *Config) {
		c.ReplaceAttr = replaceAttr
	}
}

func WithConsole(enabled bool) Option {
	return func(c *Config) {
		c.Console.Enabled = enabled
	}
}

func WithConsoleFormat(format OutputFormat) Option {
	return func(c *Config) {
		c.Console.Format = format
	}
}

func WithConsoleColor(enabled bool) Option {
	return func(c *Config) {
		c.Console.Color = enabled
	}
}

// WithConsoleFormatter sets the console formatter for logging, and automatically sets the format to FormatCustom
// The formatter string can contain the following placeholders:
// - {time}: The timestamp of the log message
// - {level}: The log level of the message
// - {message}: The log message
// - {file}: The source file where the log message was generated
// - {attrs}: Any additional attributes associated with the log message
// For example: "{time} [{level}] {file} {message} {attrs}"
func WithConsoleFormatter(formatter string) Option {
	return func(c *Config) {
		c.Console.Format = FormatCustom
		c.Console.Formatter = formatter
	}
}

func WithFile(enabled bool) Option {
	return func(c *Config) {
		c.File.Enabled = enabled
	}
}

func WithFilePath(path string) Option {
	return func(c *Config) {
		c.File.Path = path
		c.File.Enabled = true
	}
}

func WithFileFormat(format OutputFormat) Option {
	return func(c *Config) {
		c.File.Format = format
	}
}

func WithFileFormatter(formatter string) Option {
	return func(c *Config) {
		c.File.Format = FormatCustom
		c.File.Formatter = formatter
	}
}

// WithMaxSizeMB sets the maximum size of the log file in megabytes
func WithMaxSizeMB(maxSizeMB int) Option {
	return func(c *Config) {
		c.File.MaxSizeMB = maxSizeMB
	}
}

// WithRetentionDays sets the number of days to retain log files
func WithRetentionDays(retentionDays int) Option {
	return func(c *Config) {
		c.File.RetentionDays = retentionDays
	}
}

// WithFormat sets the format of the log message for both console and file logging
func WithFormat(format OutputFormat) Option {
	return func(c *Config) {
		c.Console.Format = format
		c.File.Format = format
	}
}

// WithFormatter sets the formatter for both console and file logging,
// and automatically sets the format to FormatCustom
func WithFormatter(formatter string) Option {
	return func(c *Config) {
		c.Console.Format = FormatCustom
		c.Console.Formatter = formatter
		c.File.Format = FormatCustom
		c.File.Formatter = formatter
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

	// Set default format if not provided
	if cfg.Console.Format == "" {
		cfg.Console.Format = DefaultFormat
	}
	if cfg.File.Format == "" {
		cfg.File.Format = DefaultFormat
	}

	// Validate format
	if !isValidFormat(cfg.Console.Format) {
		return fmt.Errorf("unsupported console format: %s (must be one of: text, json, custom)", cfg.Console.Format)
	}
	if !isValidFormat(cfg.File.Format) {
		return fmt.Errorf("unsupported file format: %s (must be one of: text, json, custom)", cfg.File.Format)
	}

	// Validate file configuration
	if cfg.File.Enabled {
		if cfg.File.Path == "" {
			return fmt.Errorf("file logging enabled but Path is empty")
		}

		// Create the log directory if it doesn't exist
		dir := filepath.Dir(cfg.File.Path)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("unable to create log directory %s: %w", dir, err)
			}
		} else if err != nil {
			return fmt.Errorf("error checking log directory %s: %w", dir, err)
		}

		if cfg.File.MaxSizeMB <= 0 {
			cfg.File.MaxSizeMB = DefaultMaxSizeMB
		}

		if cfg.File.RetentionDays <= 0 {
			cfg.File.RetentionDays = DefaultRetentionDays
		}
	}

	// Make sure at least one logging destination is enabled
	if !cfg.Console.Enabled && !cfg.File.Enabled {
		return fmt.Errorf("neither console nor file logging is enabled")
	}

	// Set default formatter if custom format is selected but no formatter is provided
	if cfg.Console.Format == FormatCustom && cfg.Console.Formatter == "" {
		cfg.Console.Formatter = DefaultFormatter
	}
	if cfg.File.Format == FormatCustom && cfg.File.Formatter == "" {
		cfg.File.Formatter = DefaultFormatter
	}

	return nil
}

func isValidFormat(format OutputFormat) bool {
	return format == FormatText || format == FormatJSON || format == FormatCustom
}
