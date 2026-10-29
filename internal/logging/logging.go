package logging

import (
	"context"
	"io"
	"log/slog"

	"github.com/rejisterjack/keywarden/internal/redact"
)

// New returns a JSON logger that scrubs secret-shaped values before they
// are written.
func New(w io.Writer) *slog.Logger {
	base := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(&Handler{next: base})
}

// Handler redacts log messages and attributes.
type Handler struct {
	next slog.Handler
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	nr := slog.NewRecord(r.Time, r.Level, redact.String(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		nr.AddAttrs(redactAttr(a))
		return true
	})
	return h.next.Handle(ctx, nr)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		clean[i] = redactAttr(a)
	}
	return &Handler{next: h.next.WithAttrs(clean)}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{next: h.next.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	a.Value = a.Value.Resolve()
	if redact.SensitiveKey(a.Key) {
		return slog.String(a.Key, "[redacted]")
	}
	switch a.Value.Kind() {
	case slog.KindString:
		return slog.String(a.Key, redact.String(a.Value.String()))
	case slog.KindGroup:
		group := a.Value.Group()
		out := make([]any, 0, len(group)*2)
		for _, g := range group {
			clean := redactAttr(g)
			out = append(out, clean.Key, clean.Value)
		}
		return slog.Group(a.Key, out...)
	default:
		return a
	}
}
