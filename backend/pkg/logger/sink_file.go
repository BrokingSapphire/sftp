package logger

import (
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// FileSinkConfig configures the rotating-file sink.
type FileSinkConfig struct {
	Path       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	Compress   bool
}

// FileSink writes logs to a size-rotated, compressed file. Suitable for the
// long-retention audit/compliance log required on-premise.
type FileSink struct {
	level zapcore.Level
	w     *lumberjack.Logger
}

func NewFileSink(cfg FileSinkConfig, level zapcore.Level) *FileSink {
	if cfg.MaxSizeMB == 0 {
		cfg.MaxSizeMB = 100
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 7
	}
	if cfg.MaxAgeDays == 0 {
		cfg.MaxAgeDays = 90
	}
	return &FileSink{
		level: level,
		w: &lumberjack.Logger{
			Filename:   cfg.Path,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		},
	}
}

func (s *FileSink) Core(enc zapcore.Encoder) zapcore.Core {
	return zapcore.NewCore(enc, zapcore.AddSync(s.w), s.level)
}

func (s *FileSink) Close() error { return s.w.Close() }
