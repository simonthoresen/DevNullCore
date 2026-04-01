package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// consoleSlogHandler is a slog.Handler that routes log records to the
// console's slogCh channel, formatted with timestamp and level prefix.
// It also forwards to a wrapped handler (the file handler) so logs go
// to both the console and the log file.
type consoleSlogHandler struct {
	ch      chan<- slogLine
	wrapped slog.Handler
	attrs   []slog.Attr
	group   string
}

// NewConsoleSlogHandler wraps an existing handler and tees records to ch.
func NewConsoleSlogHandler(ch chan<- slogLine, wrapped slog.Handler) slog.Handler {
	return &consoleSlogHandler{ch: ch, wrapped: wrapped}
}

func (h *consoleSlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.wrapped.Enabled(ctx, level)
}

func (h *consoleSlogHandler) Handle(ctx context.Context, r slog.Record) error {
	// Format for console display.
	cat := slogLevelToCategory(r.Level)
	prefix := slogLevelPrefix(r.Level)
	ts := r.Time.Format("15:04:05")

	// Build key=value pairs from attrs.
	var attrs string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key != "component" && a.Key != "pid" {
			attrs += " " + a.Key + "=" + a.Value.String()
		}
		return true
	})

	text := fmt.Sprintf("%s %s %s%s", ts, prefix, r.Message, attrs)

	select {
	case h.ch <- slogLine{cat: cat, text: text}:
	default:
		// Drop if channel is full.
	}

	// Forward to wrapped handler (file).
	return h.wrapped.Handle(ctx, r)
}

func (h *consoleSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &consoleSlogHandler{
		ch:      h.ch,
		wrapped: h.wrapped.WithAttrs(attrs),
		attrs:   append(h.attrs, attrs...),
		group:   h.group,
	}
}

func (h *consoleSlogHandler) WithGroup(name string) slog.Handler {
	return &consoleSlogHandler{
		ch:      h.ch,
		wrapped: h.wrapped.WithGroup(name),
		attrs:   h.attrs,
		group:   name,
	}
}

func slogLevelToCategory(level slog.Level) logCategory {
	switch {
	case level >= slog.LevelError:
		return catError
	case level >= slog.LevelWarn:
		return catWarn
	case level >= slog.LevelInfo:
		return catInfo
	default:
		return catDebug
	}
}

func slogLevelPrefix(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "[ERR]"
	case level >= slog.LevelWarn:
		return "[WRN]"
	case level >= slog.LevelInfo:
		return "[INF]"
	default:
		return "[DBG]"
	}
}

// InstallConsoleSlogHandler wraps the current default slog handler to also
// route records to the server's slogCh. Call this after the server is created.
func (a *Server) InstallConsoleSlogHandler() {
	existing := slog.Default().Handler()
	handler := NewConsoleSlogHandler(a.slogCh, existing)
	// Re-enable debug level for the console handler since we filter in the UI.
	wrapper := &slogMinLevelHandler{inner: handler, level: slog.LevelDebug}
	slog.SetDefault(slog.New(wrapper))
}

// slogMinLevelHandler overrides the Enabled check to allow all levels.
type slogMinLevelHandler struct {
	inner slog.Handler
	level slog.Level
}

func (h *slogMinLevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}
func (h *slogMinLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.inner.Handle(ctx, r)
}
func (h *slogMinLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &slogMinLevelHandler{inner: h.inner.WithAttrs(attrs), level: h.level}
}
func (h *slogMinLevelHandler) WithGroup(name string) slog.Handler {
	return &slogMinLevelHandler{inner: h.inner.WithGroup(name), level: h.level}
}

// Silence the "unused" warning for time import.
var _ = time.Now
