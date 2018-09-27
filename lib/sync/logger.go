package sync

import "github.com/inconshreveable/log15"

// NopLogger returns a Logger with a no-op (nil) Logger
func NopLogger() log15.Logger {
	return &nopLogger{}
}

type nopLogger struct{}

func (l *nopLogger) New(ctx ...interface{}) log15.Logger {
	return l
}

func (l nopLogger) GetHandler() log15.Handler {
	return nil
}

func (l nopLogger) SetHandler(log15.Handler)             {}
func (l nopLogger) Debug(msg string, ctx ...interface{}) {}
func (l nopLogger) Info(msg string, ctx ...interface{})  {}
func (l nopLogger) Warn(msg string, ctx ...interface{})  {}
func (l nopLogger) Error(msg string, ctx ...interface{}) {}
func (l nopLogger) Crit(msg string, ctx ...interface{})  {}
