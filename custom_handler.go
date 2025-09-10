package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// Placeholders
	PlaceholderTime    = "{time}"
	PlaceholderLevel   = "{level}"
	PlaceholderMessage = "{message}"
	PlaceholderFile    = "{file}"
	PlaceholderAttrs   = "{attrs}"

	// ANSI escape codes
	ansiReset          = "\033[0m"
	ansiFaint          = "\033[2m"
	ansiResetFaint     = "\033[22m"
	ansiBrightCyan     = "\033[96m"
	ansiBrightRed      = "\033[91m"
	ansiBrightRedFaint = "\033[91;2m"
	ansiBrightGreen    = "\033[92m"
	ansiBrightYellow   = "\033[93m"
	ansiBrightBlue     = "\033[94m"
	ansiBrightMagenta  = "\033[95m"
)

// handlerConfig stores immutable configuration data for atomic access
type handlerConfig struct {
	globalCfg  *Config
	outputCfg  outputConfig
	attrsIndex int
	groups     []string
	attrs      []slog.Attr
	opts       slog.HandlerOptions
}

type customHandler struct {
	// Lightweight mutex to protect write operations
	writeMu sync.Mutex
	out     io.Writer

	// Configuration data, accessed using atomic operations
	config atomic.Value // *handlerConfig

	// String builder pool, thread-safe
	pool *sync.Pool
}

// outputConfig interface for unified access to Console and File configurations
type outputConfig interface {
	GetFormat() OutputFormat
	GetColor() bool
	GetFormatter() string
}

// ConsoleConfig implements outputConfig interface
func (c *ConsoleConfig) GetFormat() OutputFormat {
	return c.Format
}

func (c *ConsoleConfig) GetColor() bool {
	return c.Color
}

func (c *ConsoleConfig) GetFormatter() string {
	return c.Formatter
}

// FileConfig implements outputConfig interface
func (c *FileConfig) GetFormat() OutputFormat {
	return c.Format
}

func (c *FileConfig) GetColor() bool {
	// File output does not support color
	return false
}

func (c *FileConfig) GetFormatter() string {
	return c.Formatter
}

func newCustomHandler(w io.Writer, globalCfg *Config, outputCfg outputConfig, opts *slog.HandlerOptions) (slog.Handler, error) {
	formatter := outputCfg.GetFormatter()
	if formatter == "" {
		formatter = DefaultFormatter
	}

	// Create configuration object
	cfg := &handlerConfig{
		globalCfg:  globalCfg,
		outputCfg:  outputCfg,
		attrsIndex: strings.Index(formatter, PlaceholderAttrs),
		groups:     make([]string, 0),
		attrs:      make([]slog.Attr, 0),
	}

	if opts != nil {
		cfg.opts = *opts
	} else {
		cfg.opts = slog.HandlerOptions{
			Level:       globalCfg.Level,
			AddSource:   globalCfg.AddSource,
			ReplaceAttr: globalCfg.ReplaceAttr,
		}
	}

	h := &customHandler{
		out: w,
		pool: &sync.Pool{
			New: func() any {
				return new(strings.Builder)
			},
		},
	}

	// Atomically set the configuration
	h.config.Store(cfg)

	return h, nil
}

// getConfig atomically retrieves the current configuration
func (h *customHandler) getConfig() *handlerConfig {
	return h.config.Load().(*handlerConfig)
}

func (h *customHandler) Enabled(_ context.Context, level slog.Level) bool {
	cfg := h.getConfig()
	return level >= cfg.opts.Level.Level()
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	// Lock-free access to config and formatting
	cfg := h.getConfig()

	// Add preset attributes to the record
	for _, attr := range cfg.attrs {
		r.AddAttrs(attr)
	}

	// Lock-free log formatting (CPU-intensive operation)
	builder := h.pool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		h.pool.Put(builder)
	}()

	h.formatLogLine(builder, r, cfg)
	logData := []byte(builder.String())

	// Only lock during write (I/O operation)
	h.writeMu.Lock()
	_, err := h.out.Write(logData)
	h.writeMu.Unlock()

	return err
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	// Lock-free operation: copy config and add new attributes
	oldCfg := h.getConfig()
	newCfg := &handlerConfig{
		globalCfg:  oldCfg.globalCfg,
		outputCfg:  oldCfg.outputCfg,
		attrsIndex: oldCfg.attrsIndex,
		groups:     slices.Clone(oldCfg.groups),
		attrs:      append(slices.Clone(oldCfg.attrs), attrs...),
		opts:       oldCfg.opts,
	}

	newHandler := &customHandler{
		out:  h.out,
		pool: h.pool,
	}
	newHandler.config.Store(newCfg)

	return newHandler
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	// Lock-free operation: copy config and add new group
	oldCfg := h.getConfig()
	newCfg := &handlerConfig{
		globalCfg:  oldCfg.globalCfg,
		outputCfg:  oldCfg.outputCfg,
		attrsIndex: oldCfg.attrsIndex,
		groups:     append(slices.Clone(oldCfg.groups), name),
		attrs:      slices.Clone(oldCfg.attrs),
		opts:       oldCfg.opts,
	}

	newHandler := &customHandler{
		out:  h.out,
		pool: h.pool,
	}
	newHandler.config.Store(newCfg)

	return newHandler
}

func (h *customHandler) formatLogLine(builder *strings.Builder, r slog.Record, cfg *handlerConfig) {
	// Process built-in attributes through ReplaceAttr like standard slog handlers
	rep := cfg.opts.ReplaceAttr

	// Build all the parts, applying ReplaceAttr to built-in attributes
	var timeStr, levelStr, msgStr, fileStr string

	// Handle time (built-in attribute)
	if !r.Time.IsZero() {
		timeAttr := slog.Time(slog.TimeKey, r.Time.In(cfg.globalCfg.TimeZone))
		if rep != nil {
			timeAttr = rep(nil, timeAttr) // Built-ins are not in any group
		}
		if !timeAttr.Equal(slog.Attr{}) { // Check if not removed by ReplaceAttr
			timeValue := timeAttr.Value.Any()
			if t, ok := timeValue.(time.Time); ok {
				timeStr = h.colorize(t.Format(cfg.globalCfg.TimeFormat), ansiFaint, cfg)
			} else {
				// ReplaceAttr changed the type, use the new value
				timeStr = h.colorize(fmt.Sprintf("%v", timeValue), ansiFaint, cfg)
			}
		}
	}

	// Handle level (built-in attribute)
	levelAttr := slog.Any(slog.LevelKey, r.Level)
	if rep != nil {
		levelAttr = rep(nil, levelAttr) // Built-ins are not in any group
	}
	if !levelAttr.Equal(slog.Attr{}) { // Check if not removed by ReplaceAttr
		levelValue := levelAttr.Value.Any()
		if level, ok := levelValue.(slog.Level); ok {
			levelStr = h.colorizeLevel(level, cfg)
		} else {
			// ReplaceAttr changed the type, use the new value
			levelStr = h.colorize(fmt.Sprintf("%v", levelValue), ansiBrightGreen, cfg)
		}
	}

	// Handle message (built-in attribute)
	msgAttr := slog.String(slog.MessageKey, r.Message)
	if rep != nil {
		msgAttr = rep(nil, msgAttr) // Built-ins are not in any group
	}
	if !msgAttr.Equal(slog.Attr{}) { // Check if not removed by ReplaceAttr
		msgStr = h.colorizeMessage(msgAttr.Value.String(), r.Level, cfg)
	}

	// Handle source/file (built-in attribute)
	if cfg.opts.AddSource {
		// Create source attribute like standard slog handlers
		var source *slog.Source
		if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			source = &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}
		} else {
			// Create empty source if PC is zero (like standard slog)
			source = &slog.Source{}
		}

		sourceAttr := slog.Any(slog.SourceKey, source)
		if rep != nil {
			sourceAttr = rep(nil, sourceAttr) // Built-ins are not in any group
		}
		if !sourceAttr.Equal(slog.Attr{}) { // Check if not removed by ReplaceAttr
			sourceValue := sourceAttr.Value.Any()
			if src, ok := sourceValue.(*slog.Source); ok {
				if src.File != "" {
					// Standard format: filename:function:line
					fileStr = h.colorize(fmt.Sprintf("%s:%s:%d", filepath.Base(src.File), filepath.Base(src.Function), src.Line), ansiFaint, cfg)
				}
			} else {
				// ReplaceAttr changed the type, use the new value
				fileStr = h.colorize(fmt.Sprintf("%v", sourceValue), ansiFaint, cfg)
			}
		}
	}

	// Handle user attributes
	var attrsStr string
	if cfg.attrsIndex >= 0 {
		attrBuilder := h.pool.Get().(*strings.Builder)
		defer func() {
			attrBuilder.Reset()
			h.pool.Put(attrBuilder)
		}()

		isFirst := true
		r.Attrs(func(a slog.Attr) bool {
			// Apply ReplaceAttr if configured
			if rep != nil {
				a = rep(cfg.groups, a) // User attributes use current groups
			}
			h.appendColorizedAttr(attrBuilder, a, r.Level, isFirst, cfg)
			isFirst = false
			return true
		})
		attrsStr = attrBuilder.String()
	}

	// Replace placeholders - use conditional replacement to avoid empty placeholder issues
	logLine := cfg.outputCfg.GetFormatter()

	// Replace each placeholder individually, handling empty cases
	logLine = strings.ReplaceAll(logLine, PlaceholderTime, timeStr)
	logLine = strings.ReplaceAll(logLine, PlaceholderLevel, levelStr)
	logLine = strings.ReplaceAll(logLine, PlaceholderMessage, msgStr)

	// Handle file placeholder - only replace if not empty
	if fileStr != "" {
		logLine = strings.ReplaceAll(logLine, PlaceholderFile, fileStr)
	} else {
		// Remove the placeholder and any adjacent spaces
		logLine = h.removeEmptyPlaceholder(logLine, PlaceholderFile)
	}

	// Handle attrs placeholder - only replace if not empty
	if attrsStr != "" {
		logLine = strings.ReplaceAll(logLine, PlaceholderAttrs, attrsStr)
	} else {
		// Remove the placeholder and any adjacent spaces
		logLine = h.removeEmptyPlaceholder(logLine, PlaceholderAttrs)
	}

	builder.WriteString(logLine)
	builder.WriteString("\n")
}

func (h *customHandler) colorize(s, color string, cfg *handlerConfig) string {
	if !cfg.outputCfg.GetColor() {
		return s
	}
	return color + s + ansiReset
}

func (h *customHandler) colorizeLevel(level slog.Level, cfg *handlerConfig) string {
	var color string
	switch {
	case level <= slog.LevelDebug:
		color = ansiBrightCyan
	case level <= slog.LevelInfo:
		color = ansiBrightGreen
	case level <= slog.LevelWarn:
		color = ansiBrightYellow
	case level <= slog.LevelError:
		color = ansiBrightRed
	default:
		color = ansiBrightMagenta
	}

	return h.colorize(level.String(), color, cfg)
}

func (h *customHandler) colorizeMessage(msg string, level slog.Level, cfg *handlerConfig) string {
	if level >= slog.LevelError {
		return h.colorize(msg, ansiBrightRed, cfg)
	}
	return msg
}

func (h *customHandler) appendColorizedAttr(builder *strings.Builder, a slog.Attr, level slog.Level, isFirst bool, cfg *handlerConfig) {
	if a.Equal(slog.Attr{}) {
		return
	}

	if !isFirst {
		builder.WriteByte(' ')
	}

	// Build the key with group prefixes (slog standard behavior)
	key := a.Key
	if len(cfg.groups) > 0 {
		key = strings.Join(cfg.groups, ".") + "." + a.Key
	}

	if level >= slog.LevelError && a.Key == "error" {
		builder.WriteString(h.colorize(key, ansiBrightRedFaint, cfg))
		builder.WriteString(h.colorize("=", ansiBrightRedFaint, cfg))
		builder.WriteString(h.colorize(fmt.Sprintf("%v", a.Value.Any()), ansiBrightRed, cfg))
	} else {
		builder.WriteString(h.colorize(key, ansiFaint, cfg))
		builder.WriteString(h.colorize("=", ansiFaint, cfg))
		builder.WriteString(fmt.Sprintf("%v", a.Value.Any()))
	}
}

// removeEmptyPlaceholder removes a placeholder and adjacent spaces when the placeholder is empty
func (h *customHandler) removeEmptyPlaceholder(s, placeholder string) string {
	// Pattern: try to remove " {placeholder} " -> " "
	// Pattern: try to remove " {placeholder}" -> ""
	// Pattern: try to remove "{placeholder} " -> ""

	// Try different patterns of space-placeholder-space combinations
	patterns := []struct {
		search  string
		replace string
	}{
		{" " + placeholder + " ", " "}, // space-placeholder-space -> space
		{" " + placeholder, ""},        // space-placeholder -> nothing
		{placeholder + " ", ""},        // placeholder-space -> nothing
		{placeholder, ""},              // just placeholder -> nothing
	}

	result := s
	for _, pattern := range patterns {
		if strings.Contains(result, pattern.search) {
			result = strings.ReplaceAll(result, pattern.search, pattern.replace)
			break // Only apply the first matching pattern
		}
	}

	return result
}
