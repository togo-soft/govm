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

var _ slog.Handler = (*ConsoleHandler)(nil)

// ConsoleHandler formats log output with checkmark/cross indicators.
type ConsoleHandler struct {
	w io.Writer
}

func (c *ConsoleHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= slog.LevelDebug
}

func (c *ConsoleHandler) Handle(ctx context.Context, record slog.Record) error {
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

func (c *ConsoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return c
}

func (c *ConsoleHandler) WithGroup(name string) slog.Handler {
	return c
}

// Setup configures the global slog logger with console and file output.
// Returns a cleanup function that closes the log file.
func Setup(dir, name string) func() {
	logPath := filepath.Join(dir, name)
	handle, err := os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		slog.Error("open log file failed, using default logger", "reason", err)
		return func() {}
	}

	opts := &slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug}
	l := slog.New(slogmulti.Fanout(
		&ConsoleHandler{w: os.Stdout},
		slog.NewTextHandler(handle, opts),
	))

	slog.SetDefault(l)

	return func() {
		_ = handle.Close()
	}
}
