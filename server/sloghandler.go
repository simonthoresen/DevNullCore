package server

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
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
	// Always forward to the wrapped handler (file) first.
	err := h.wrapped.Handle(ctx, r)

	// Don't route to the console channel if we're inside a View/Render call.
	// Sending to the channel triggers Update → View → more slog calls → loop.
	if isRenderPath() {
		return err
	}

	cat := slogLevelToCategory(r.Level)
	prefix := slogLevelPrefix(r.Level)
	ts := r.Time.Format("15:04:05")

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
	}

	return err
}

// isRenderPath checks the call stack for View or Render methods.
// If found, we're inside the render cycle and must not send to the console
// channel (which would trigger Update → View → slog → feedback loop).
func isRenderPath() bool {
	var pcs [16]uintptr
	n := runtime.Callers(3, pcs[:]) // skip Callers, isRenderPath, Handle
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// Check for BubbleTea View() or NC widget Render() methods.
		fn := frame.Function
		if strings.HasSuffix(fn, ".View") ||
			strings.HasSuffix(fn, ".Render") {
			return true
		}
		if !more {
			break
		}
	}
	return false
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
// route records to the server's slogCh. Call after server creation.
func (a *Server) InstallConsoleSlogHandler() {
	existing := slog.Default().Handler()
	handler := NewConsoleSlogHandler(a.slogCh, existing)
	slog.SetDefault(slog.New(handler))
}
