package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	slogmulti "github.com/samber/slog-multi"
)

type Logger struct {
	logger        *slog.Logger
	defaultLogger *slog.Logger
	fileHandler   *os.File
}

var _ slog.Handler = (*ConsoleLogger)(nil)

type ConsoleLogger struct {
	w io.Writer
}

func (c *ConsoleLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

func (c *ConsoleLogger) Handle(ctx context.Context, record slog.Record) error {
	var flag = "✓"
	if record.Level > slog.LevelInfo {
		flag = "✗"
	}
	_, _ = fmt.Fprintf(c.w, "%s %s", flag, record.Message)
	record.Attrs(func(attr slog.Attr) bool {
		_, _ = fmt.Fprintf(c.w, " [%s]=\"%v\"", attr.Key, attr.Value)
		return true
	})

	_, _ = fmt.Fprintln(c.w)
	return nil
}

func (c *ConsoleLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c
}

func (c *ConsoleLogger) WithGroup(name string) slog.Handler {
	return c
}

func NewLogger(wd, name string) *Logger {
	var log = new(Logger)
	log.defaultLogger = slog.Default()

	logName := filepath.Join(wd, name)
	handle, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		log.defaultLogger.Error("open log file failed, use default logger", "reason", err)
		return log
	}
	log.fileHandler = handle
	defaultOpts := &slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug}
	log.logger = slog.New(slogmulti.Fanout(
		&ConsoleLogger{w: os.Stdout},
		slog.NewTextHandler(handle, defaultOpts),
	))

	slog.SetDefault(log.logger)

	return log
}

func (log *Logger) Close() {
	if log.fileHandler == nil {
		return
	}
	_ = log.fileHandler.Close()
}

func (log *Logger) Debug(msg string, args ...interface{}) {
	slog.Debug(msg, args...)
}

func (log *Logger) Info(msg string, args ...interface{}) {
	slog.Info(msg, args...)
}

func (log *Logger) Warn(msg string, args ...interface{}) {
	slog.Warn(msg, args...)
}

func (log *Logger) Error(msg string, args ...interface{}) {
	slog.Error(msg, args...)
}
