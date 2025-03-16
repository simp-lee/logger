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

type customHandler struct {
	mu         sync.Mutex
	out        io.Writer
	cfg        *Config
	attrsIndex int
	pool       *sync.Pool
	groups     []string
	attrs      []slog.Attr
	opts       slog.HandlerOptions
}

func newCustomHandler(w io.Writer, cfg *Config, opts *slog.HandlerOptions) (slog.Handler, error) {
	h := &customHandler{
		out:        w,
		cfg:        cfg,
		attrsIndex: strings.Index(cfg.Formatter, PlaceholderAttrs),
		pool: &sync.Pool{
			New: func() interface{} {
				return new(strings.Builder)
			},
		},
		attrs: make([]slog.Attr, 0),
	}

	if opts != nil {
		h.opts = *opts
	} else {
		h.opts = slog.HandlerOptions{
			Level:       cfg.Level,
			AddSource:   cfg.AddSource,
			ReplaceAttr: cfg.ReplaceAttr,
		}
	}

	return h, nil
}

func (h *customHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *customHandler) Handle(ctx context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	builder := h.pool.Get().(*strings.Builder)
	defer func() {
		builder.Reset()
		h.pool.Put(builder)
	}()

	for _, attr := range h.attrs {
		r.AddAttrs(attr)
	}

	h.formatLogLine(builder, r)

	_, err := h.out.Write([]byte(builder.String()))
	return err
}

func (h *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := h.clone()
	newHandler.attrs = append(slices.Clone(h.attrs), attrs...)
	return newHandler
}

func (h *customHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newHandler := h.clone()
	newHandler.groups = append(slices.Clone(h.groups), name)
	return newHandler
}

func (h *customHandler) clone() *customHandler {
	return &customHandler{
		out:        h.out,
		cfg:        h.cfg,
		attrsIndex: h.attrsIndex,
		pool:       h.pool,
		groups:     slices.Clone(h.groups),
		attrs:      slices.Clone(h.attrs),
		opts:       h.opts,
	}
}

func (h *customHandler) formatLogLine(builder *strings.Builder, r slog.Record) {
	timeStr := h.colorize(r.Time.In(h.cfg.TimeZone).Format(h.cfg.TimeFormat), ansiFaint)
	levelStr := h.colorizeLevel(r.Level)

	// Handle group prefix
	var msg string
	if len(h.groups) > 0 {
		groupPrefix := strings.Join(h.groups, ".")
		msg = fmt.Sprintf("%s: %s", groupPrefix, r.Message)
	} else {
		msg = r.Message
	}
	msg = h.colorizeMessage(msg, r.Level)

	// Handle file
	var fileStr string
	if h.opts.AddSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			fileStr = h.colorize(fmt.Sprintf("%s:%s:%d", filepath.Base(f.File), filepath.Base(f.Function), f.Line), ansiFaint)
		}
	}

	// Handle attributes
	var attrsStr string
	if h.attrsIndex >= 0 {
		attrBuilder := h.pool.Get().(*strings.Builder)
		defer func() {
			attrBuilder.Reset()
			h.pool.Put(attrBuilder)
		}()

		isFirst := true
		r.Attrs(func(a slog.Attr) bool {
			h.appendColorizedAttr(attrBuilder, a, r.Level, isFirst)
			isFirst = false
			return true
		})
		attrsStr = attrBuilder.String()
	}

	// Replace placeholders
	logLine := strings.NewReplacer(
		PlaceholderTime, timeStr,
		PlaceholderLevel, levelStr,
		PlaceholderMessage, msg,
		PlaceholderFile, fileStr,
		PlaceholderAttrs, attrsStr,
	).Replace(h.cfg.Formatter)

	// Remove extra spaces and add newline
	builder.WriteString(strings.Join(strings.Fields(logLine), " "))
	builder.WriteString("\n")
}

func (h *customHandler) colorize(s, color string) string {
	if !h.cfg.Color {
		return s
	}
	return color + s + ansiReset
}

func (h *customHandler) colorizeLevel(level slog.Level) string {
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

	return h.colorize(level.String(), color)
}

func (h *customHandler) colorizeMessage(msg string, level slog.Level) string {
	if level >= slog.LevelError {
		return h.colorize(msg, ansiBrightRed)
	}
	return msg
}

func (h *customHandler) appendColorizedAttr(builder *strings.Builder, a slog.Attr, level slog.Level, isFirst bool) {
	if a.Equal(slog.Attr{}) {
		return
	}

	if !isFirst {
		builder.WriteByte(' ')
	}

	if level >= slog.LevelError && a.Key == "error" {
		builder.WriteString(h.colorize(a.Key, ansiBrightRedFaint))
		builder.WriteString(h.colorize("=", ansiBrightRedFaint))
		builder.WriteString(h.colorize(fmt.Sprintf("%v", a.Value.Any()), ansiBrightRed))
	} else {
		builder.WriteString(h.colorize(a.Key, ansiFaint))
		builder.WriteString(h.colorize("=", ansiFaint))
		builder.WriteString(fmt.Sprintf("%v", a.Value.Any()))
	}
}
