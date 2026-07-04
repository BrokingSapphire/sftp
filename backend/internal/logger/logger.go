// Package logger provides a configured Zap structured logger.
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New builds a *zap.Logger from level ("debug"|"info"|"warn"|"error")
// and format ("json"|"console").
func New(level, format string) (*zap.Logger, error) {
	lvl := zapcore.InfoLevel
	_ = lvl.UnmarshalText([]byte(level))

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Encoding:         encoding(format),
		EncoderConfig:    encCfg,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if format == "console" {
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	return cfg.Build()
}

func encoding(format string) string {
	if format == "console" {
		return "console"
	}
	return "json"
}
