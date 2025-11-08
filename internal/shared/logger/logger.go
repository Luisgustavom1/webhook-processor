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

func (l *Logger) Debug(msg string, args ...any) {
	l.s.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.s.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.s.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.s.Error(msg, args...)
}

func (l *Logger) SetAsDefaultForPackage() {
	slog.SetDefault(l.s)
}
