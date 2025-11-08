package logger

import (
	"log/slog"
	"os"
)

var LevelMap = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

type Logger struct {
	level  slog.Level
	prefix string
	s      *slog.Logger
}

type NewLoggerOptions struct {
	Level  string
	Prefix string
}

func NewLogger(options *NewLoggerOptions) *Logger {
	l := LevelMap[options.Level]
	s := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
	prefix := options.Prefix

	if prefix != "" {
		s.WithGroup(prefix)
	}

	return &Logger{
		level:  l,
		prefix: prefix,
		s:      s,
	}
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func (l *Logger) SetAsDefaultForPackage() {
	slog.SetDefault(l.s)
}
