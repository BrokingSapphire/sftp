package logger

import (
	"os"

	"go.uber.org/zap/zapcore"
)

// StdoutSink writes logs to standard output.
type StdoutSink struct {
	level zapcore.Level
}

func NewStdoutSink(level zapcore.Level) *StdoutSink {
	return &StdoutSink{level: level}
}

func (s *StdoutSink) Core(enc zapcore.Encoder) zapcore.Core {
	return zapcore.NewCore(enc, zapcore.AddSync(os.Stdout), s.level)
}

func (s *StdoutSink) Close() error { return nil }
