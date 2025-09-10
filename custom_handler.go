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

// TokenType represents the type of a template token
type TokenType int

const (
	TokenTypeText TokenType = iota
	TokenTypeTime
	TokenTypeLevel
	TokenTypeMessage
	TokenTypeFile
	TokenTypeAttrs
)

// Token represents a parsed template component
type Token struct {
	Type TokenType
	Text string // For static text tokens
}

// ParsedTemplate holds the pre-parsed template tokens
type ParsedTemplate struct {
	tokens []Token
}

// handlerConfig stores immutable configuration data for atomic access
type handlerConfig struct {
	globalCfg      *Config
	outputCfg      outputConfig
	attrsIndex     int
	groups         []string
	attrs          []slog.Attr
	opts           slog.HandlerOptions
	parsedTemplate *ParsedTemplate // Pre-parsed template for efficient formatting
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

// parseTemplate parses a format template into tokens for efficient rendering
func parseTemplate(template string) *ParsedTemplate {
	if template == "" {
		template = DefaultFormatter
	}

	var tokens []Token
	remaining := template

	for len(remaining) > 0 {
		// Find the next placeholder
		nextPlaceholder := -1
		var placeholderType TokenType
		var placeholderLen int

		// Check for each placeholder type
		placeholders := []struct {
			text      string
			tokenType TokenType
		}{
			{PlaceholderTime, TokenTypeTime},
			{PlaceholderLevel, TokenTypeLevel},
			{PlaceholderMessage, TokenTypeMessage},
			{PlaceholderFile, TokenTypeFile},
			{PlaceholderAttrs, TokenTypeAttrs},
		}

		for _, p := range placeholders {
			if idx := strings.Index(remaining, p.text); idx != -1 {
				if nextPlaceholder == -1 || idx < nextPlaceholder {
					nextPlaceholder = idx
					placeholderType = p.tokenType
					placeholderLen = len(p.text)
				}
			}
		}

		if nextPlaceholder == -1 {
			// No more placeholders, add remaining text as static token
			if len(remaining) > 0 {
				tokens = append(tokens, Token{Type: TokenTypeText, Text: remaining})
			}
			break
		}

		// Add static text before placeholder (if any)
		if nextPlaceholder > 0 {
			tokens = append(tokens, Token{Type: TokenTypeText, Text: remaining[:nextPlaceholder]})
		}

		// Add placeholder token
		tokens = append(tokens, Token{Type: placeholderType})

		// Move to after the placeholder
		remaining = remaining[nextPlaceholder+placeholderLen:]
	}

	return &ParsedTemplate{tokens: tokens}
}

func newCustomHandler(w io.Writer, globalCfg *Config, outputCfg outputConfig, opts *slog.HandlerOptions) (slog.Handler, error) {
	formatter := outputCfg.GetFormatter()
	if formatter == "" {
		formatter = DefaultFormatter
	}

	// Parse template at startup time for efficient formatting
	parsedTemplate := parseTemplate(formatter)

	// Create configuration object
	cfg := &handlerConfig{
		globalCfg:      globalCfg,
		outputCfg:      outputCfg,
		attrsIndex:     strings.Index(formatter, PlaceholderAttrs),
		groups:         make([]string, 0),
		attrs:          make([]slog.Attr, 0),
		parsedTemplate: parsedTemplate,
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
		globalCfg:      oldCfg.globalCfg,
		outputCfg:      oldCfg.outputCfg,
		attrsIndex:     oldCfg.attrsIndex,
		groups:         slices.Clone(oldCfg.groups),
		attrs:          append(slices.Clone(oldCfg.attrs), attrs...),
		opts:           oldCfg.opts,
		parsedTemplate: oldCfg.parsedTemplate, // Share the parsed template
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
		globalCfg:      oldCfg.globalCfg,
		outputCfg:      oldCfg.outputCfg,
		attrsIndex:     oldCfg.attrsIndex,
		groups:         append(slices.Clone(oldCfg.groups), name),
		attrs:          slices.Clone(oldCfg.attrs),
		opts:           oldCfg.opts,
		parsedTemplate: oldCfg.parsedTemplate, // Share the parsed template
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

	// Pre-compute all the parts that might be needed
	var timeStr, levelStr, msgStr, fileStr, attrsStr string

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

	// Use parsed template for efficient formatting
	h.renderTemplate(builder, cfg.parsedTemplate, timeStr, levelStr, msgStr, fileStr, attrsStr)
	builder.WriteString("\n")
}

// renderTemplate efficiently renders the parsed template by iterating through tokens
func (h *customHandler) renderTemplate(builder *strings.Builder, template *ParsedTemplate, timeStr, levelStr, msgStr, fileStr, attrsStr string) {
	tokens := template.tokens
	for i, token := range tokens {
		switch token.Type {
		case TokenTypeText:
			// Handle text tokens, but be smart about spaces around empty placeholders
			text := token.Text

			// If this is a space before an empty placeholder, and we're followed by another space, skip one space
			if text == " " && i+2 < len(tokens) {
				nextToken := tokens[i+1]
				nextNextToken := tokens[i+2]

				// Check if next token is empty and followed by space
				isEmpty := false
				switch nextToken.Type {
				case TokenTypeTime:
					isEmpty = timeStr == ""
				case TokenTypeLevel:
					isEmpty = levelStr == ""
				case TokenTypeMessage:
					isEmpty = msgStr == ""
				case TokenTypeFile:
					isEmpty = fileStr == ""
				case TokenTypeAttrs:
					isEmpty = attrsStr == ""
				}

				// If next placeholder is empty and followed by space, skip this space
				if isEmpty && nextNextToken.Type == TokenTypeText && strings.HasPrefix(nextNextToken.Text, " ") {
					continue
				}
			}

			builder.WriteString(text)
		case TokenTypeTime:
			if timeStr != "" {
				builder.WriteString(timeStr)
			}
		case TokenTypeLevel:
			if levelStr != "" {
				builder.WriteString(levelStr)
			}
		case TokenTypeMessage:
			if msgStr != "" {
				builder.WriteString(msgStr)
			}
		case TokenTypeFile:
			if fileStr != "" {
				builder.WriteString(fileStr)
			}
		case TokenTypeAttrs:
			if attrsStr != "" {
				builder.WriteString(attrsStr)
			}
		}
	}
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
		fmt.Fprintf(builder, "%v", a.Value.Any())
	}
}
