package sloggorm

import (
	"context"
	"cpamgt/internal/pkg/log"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

var ctxLoggerKey = log.GetCtxLoggerKey()

type Logger struct {
	SlogLogger                *slog.Logger
	SlowThreshold             time.Duration
	Colorful                  bool
	IgnoreRecordNotFoundError bool
	ParameterizedQueries      bool
	LogLevel                  gormlogger.LogLevel
}

func New(slogLogger *slog.Logger) gormlogger.Interface {
	return &Logger{
		SlogLogger:                slogLogger,
		LogLevel:                  gormlogger.Warn,
		SlowThreshold:             100 * time.Millisecond,
		Colorful:                  false,
		IgnoreRecordNotFoundError: false,
		ParameterizedQueries:      false,
	}
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l *Logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.log(ctx, slog.LevelInfo, fmt.Sprintf(msg, data...))
	}
}

// Warn print warn messages
func (l *Logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.log(ctx, slog.LevelWarn, fmt.Sprintf(msg, data...))
	}
}

// Error print error messages
func (l *Logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.log(ctx, slog.LevelError, fmt.Sprintf(msg, data...))
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	elapsedStr := fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)

	switch {
	case err != nil && l.LogLevel >= gormlogger.Error && (!errors.Is(err, gormlogger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		l.log(ctx, slog.LevelError, "trace",
			slog.Any("error", err),
			slog.String("elapsed", elapsedStr),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= gormlogger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		l.log(ctx, slog.LevelWarn, "trace",
			slog.String("slow", slowLog),
			slog.String("elapsed", elapsedStr),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	case l.LogLevel == gormlogger.Info:
		sql, rows := fc()
		l.log(ctx, slog.LevelInfo, "trace",
			slog.String("elapsed", elapsedStr),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	}
}

var (
	gormPackage = filepath.Join("gorm.io", "gorm")
)

// log 核心打印方法，处理上下文提取和正确的代码行号（Caller 跳过）
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logger := l.SlogLogger

	// 1. 从 context 中提取 Logger
	if ctx != nil {
		if ctxLogger, ok := ctx.Value(ctxLoggerKey).(*slog.Logger); ok {
			logger = ctxLogger
		}
	}

	// 2. 检查日志级别是否启用，避免不必要的性能开销
	if !logger.Enabled(ctx, level) {
		return
	}

	// 3. 寻找真实调用者的 Program Counter (PC)，跳过 gorm 内部代码
	var pc uintptr
	for i := 2; i < 15; i++ {
		rpc, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}
		if strings.HasSuffix(file, "_test.go") || strings.Contains(file, "gorm.io/gorm") {
			continue
		}
		pc = rpc
		break
	}

	// 4. 构建并发送 slog.Record
	r := slog.NewRecord(time.Now(), level, msg, pc)
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}
