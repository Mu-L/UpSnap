package logging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

var consoleHandler slog.Handler

// ConfigureConsole sets console output for UpSnap logs. PocketBase persistence
// remains controlled by PocketBase's built-in log settings.
func ConfigureConsole(level string) error {
	level = strings.TrimSpace(level)
	if level == "" {
		level = "INFO"
	}

	if strings.EqualFold(level, "OFF") {
		consoleHandler = nil
		return nil
	}

	var parsed slog.Level
	if err := parsed.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		return fmt.Errorf("invalid UPSNAP_LOG_LEVEL %q (expected DEBUG, INFO, WARN, ERROR, or OFF)", level)
	}

	consoleHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource:   true,
		Level:       parsed,
		ReplaceAttr: replaceSource,
	})
	return nil
}

func replaceSource(_ []string, attr slog.Attr) slog.Attr {
	if attr.Key != slog.SourceKey {
		return attr
	}

	source, ok := attr.Value.Any().(*slog.Source)
	if !ok {
		return attr
	}

	file := strings.ReplaceAll(source.File, "\\", "/")
	if index := strings.LastIndex(file, "/backend/"); index >= 0 {
		file = file[index+1:]
	}

	return slog.String(attr.Key, file+":"+strconv.Itoa(source.Line))
}

// Logger writes through PocketBase's logger and mirrors UpSnap logs to stdout.
func Logger(app core.App) *slog.Logger {
	handlers := []slog.Handler{app.Logger().Handler()}
	if consoleHandler != nil && !app.IsDev() {
		handlers = append(handlers, consoleHandler)
	}
	return slog.New(multiHandler(handlers))
}

type multiHandler []slog.Handler

func (h multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range h {
		if handler.Enabled(ctx, record.Level) {
			if err := handler.Handle(ctx, record.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (h multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make(multiHandler, len(h))
	for i, handler := range h {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return handlers
}

func (h multiHandler) WithGroup(name string) slog.Handler {
	handlers := make(multiHandler, len(h))
	for i, handler := range h {
		handlers[i] = handler.WithGroup(name)
	}
	return handlers
}
