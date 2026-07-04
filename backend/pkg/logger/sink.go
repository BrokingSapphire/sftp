package logger

import "go.uber.org/zap/zapcore"

// Sink is a single log destination.
type Sink interface {
	Core(enc zapcore.Encoder) zapcore.Core
	Close() error
}
