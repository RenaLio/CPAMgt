package log

import (
	"context"
	"cpamgt/internal/config"
	"log/slog"
	"os"
	"time"
)

type ctxLoggerKeyType struct{}

var ctxLoggerKey = ctxLoggerKeyType{}

type Logger struct {
	*slog.Logger
}

func NewLog(conf *config.Config) *Logger {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				if t, ok := attr.Value.Any().(time.Time); ok {
					attr.Value = slog.StringValue(t.UTC().Format(time.RFC3339))
				}
			}
			return attr
		},
	}
	//handler := slog.NewJSONHandler(os.Stdout, opts)
	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	return &Logger{logger}
}

func (l *Logger) Inject(ctx context.Context, fields ...any) context.Context {
	return context.WithValue(ctx, ctxLoggerKey, l.FromContext(ctx).With(fields...))
}

func (l *Logger) FromContext(ctx context.Context) *Logger {
	logger, ok := ctx.Value(ctxLoggerKey).(*slog.Logger)
	if !ok {
		return l
	}
	return &Logger{logger}
}
