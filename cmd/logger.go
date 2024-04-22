package cmd

import (
	"context"
	"log/slog"
)

func NewLoggerWriter(logger *slog.Logger, level slog.Level) *LoggerWriter {
	return &LoggerWriter{
		logger: logger,
		level:  level,
		ctx:    context.Background(),
	}
}

func NewLoggerWriterWithContext(logger *slog.Logger, level slog.Level, ctx context.Context) *LoggerWriter {
	return &LoggerWriter{
		logger: logger,
		level:  level,
		ctx:    ctx,
	}
}

type LoggerWriter struct {
	logger *slog.Logger
	level  slog.Level
	ctx    context.Context
}

func (w *LoggerWriter) Write(p []byte) (n int, err error) {
	w.logger.Log(context.Background(), w.level, string(p))
	return len(p), nil
}
