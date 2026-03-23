package log

import (
	"context"
	"cpamgt/internal/config"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxLoggerKeyType struct{}

var ctxLoggerKey = ctxLoggerKeyType{}

func GetCtxLoggerKey() ctxLoggerKeyType {
	return ctxLoggerKey
}

type Logger struct {
	*slog.Logger
}

func NewLog(conf *config.Config) *Logger {
	var level slog.Level
	switch conf.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				if t, ok := attr.Value.Any().(time.Time); ok {
					attr.Value = slog.StringValue(t.UTC().Format(time.RFC3339))
				}
			}
			return attr
		},
	}

	errOpt := *opts
	errOpt.Level = slog.LevelError

	// 提取一个生成文件 Writer 的闭包函数，按需调用（避免 console 模式下创建文件）
	initFileWriters := func() (io.WriteCloser, io.WriteCloser) {
		if conf.Log.LogPath != "" {
			_ = os.MkdirAll(conf.Log.LogPath, 0o755)
		}
		logWriter := getLoggerWriter(&fileOutputOption{
			Filename:   filepath.Join(conf.Log.LogPath, conf.Log.FileName), // 规范化路径拼接
			MaxSize:    conf.Log.MaxSize,
			MaxBackups: conf.Log.MaxBackups,
			MaxAge:     conf.Log.MaxAge,
			Compress:   conf.Log.Compress,
		})

		errorLogWriter := getLoggerWriter(&fileOutputOption{
			Filename:   filepath.Join(conf.Log.LogPath, conf.Log.ErrorFileName), // 规范化路径拼接
			MaxSize:    conf.Log.MaxSize,
			MaxBackups: conf.Log.MaxBackups,
			MaxAge:     conf.Log.MaxAge,
			Compress:   conf.Log.Compress,
		})
		return logWriter, errorLogWriter
	}

	handlers := make([]slog.Handler, 0)

	addFileEncodingHandler := func() {
		logWriter, errorLogWriter := initFileWriters() // 按需初始化
		if conf.Log.FileEncoding == "json" {
			handlers = append(handlers, slog.NewJSONHandler(logWriter, opts))
			handlers = append(handlers, slog.NewJSONHandler(errorLogWriter, &errOpt))
		} else {
			handlers = append(handlers, slog.NewTextHandler(logWriter, opts))
			handlers = append(handlers, slog.NewTextHandler(errorLogWriter, &errOpt))
		}
	}
	addConsoleEncodingHandler := func() {
		if conf.Log.ConsoleEncoding == "json" {
			handlers = append(handlers, slog.NewJSONHandler(os.Stdout, opts))
		} else {
			handlers = append(handlers, slog.NewTextHandler(os.Stdout, opts))
		}
	}

	switch conf.Log.Mode {
	case "console":
		addConsoleEncodingHandler()
	case "file":
		addFileEncodingHandler()
	default:
		addConsoleEncodingHandler()
		addFileEncodingHandler()
	}

	var coreHandler slog.Handler
	if len(handlers) == 1 {
		// 优化：如果只有一个 handler（比如单纯的 console），直接用即可
		coreHandler = handlers[0]
	} else {
		coreHandler = slog.NewMultiHandler(handlers...)
	}
	logger := slog.New(coreHandler)
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

type fileOutputOption struct {
	Filename   string // 日志文件路径
	MaxSize    int    // 每个日志文件保存的最大尺寸 单位：M
	MaxBackups int    // 日志文件最多保存多少个备份
	MaxAge     int    // 文件最多保存多少天
	Compress   bool   // 是否压缩
}

func getLoggerWriter(option *fileOutputOption) io.WriteCloser {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   option.Filename,   // 日志文件路径
		MaxSize:    option.MaxSize,    // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: option.MaxBackups, // 日志文件最多保存多少个备份
		MaxAge:     option.MaxAge,     // 文件最多保存多少天
		Compress:   option.Compress,   // 是否压缩
	}
	return lumberJackLogger
}
