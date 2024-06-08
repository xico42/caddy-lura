package lura

import (
	"github.com/luraproject/lura/v2/logging"
	"go.uber.org/zap"
)

type caddyLogger struct {
	logger *zap.SugaredLogger
}

func newLogger(l *zap.Logger) logging.Logger {
	return &caddyLogger{logger: l.Sugar()}
}

func (c *caddyLogger) Debug(v ...interface{}) {
	c.logger.Debug(v...)
}

func (c *caddyLogger) Info(v ...interface{}) {
	c.logger.Info(v...)
}

func (c *caddyLogger) Warning(v ...interface{}) {
	c.logger.Warn(v...)
}

func (c *caddyLogger) Error(v ...interface{}) {
	c.logger.Error(v...)
}

func (c *caddyLogger) Critical(v ...interface{}) {
	c.logger.Panic(v...)
}

func (c *caddyLogger) Fatal(v ...interface{}) {
	c.logger.Fatal(v...)
}
