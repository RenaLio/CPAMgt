package log

import (
	"context"
	"log/slog"
)

type MultiHandler struct {
	handlers []slog.Handler
}

func NewMultiHandler(h ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: h}
}

func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r)
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var hs []slog.Handler
	for _, h := range m.handlers {
		hs = append(hs, h.WithAttrs(attrs))
	}
	return NewMultiHandler(hs...)
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	var hs []slog.Handler
	for _, h := range m.handlers {
		hs = append(hs, h.WithGroup(name))
	}
	return NewMultiHandler(hs...)
}
