package database

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// gormZap adapts a *zap.Logger to the GORM logger interface.
type gormZap struct {
	log   *zap.Logger
	level gormlogger.LogLevel
}

func newGormLogger(log *zap.Logger, level gormlogger.LogLevel) gormlogger.Interface {
	return &gormZap{log: log.Named("gorm"), level: level}
}

func (g *gormZap) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &gormZap{log: g.log, level: level}
}

func (g *gormZap) Info(_ context.Context, msg string, data ...interface{}) {
	if g.level >= gormlogger.Info {
		g.log.Sugar().Infof(msg, data...)
	}
}

func (g *gormZap) Warn(_ context.Context, msg string, data ...interface{}) {
	if g.level >= gormlogger.Warn {
		g.log.Sugar().Warnf(msg, data...)
	}
}

func (g *gormZap) Error(_ context.Context, msg string, data ...interface{}) {
	if g.level >= gormlogger.Error {
		g.log.Sugar().Errorf(msg, data...)
	}
}

func (g *gormZap) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound) && g.level >= gormlogger.Error:
		g.log.Error("query failed", append(fields, zap.Error(err))...)
	case elapsed > 200*time.Millisecond && g.level >= gormlogger.Warn:
		g.log.Warn("slow query", fields...)
	case g.level >= gormlogger.Info:
		g.log.Debug("query", fields...)
	}
}
