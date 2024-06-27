package kgen

import "log"

type Logger interface {
	Infof(msg string, args ...any)
	Warnf(msg string, args ...any)
	Panicf(msg string, args ...any)
}

type CustomLoggerOptions struct {
	InfofFn  func(msg string, args ...any)
	WarnfFn  func(msg string, args ...any)
	PanicfFn func(msg string, args ...any)
}

type customLogger struct {
	opts *CustomLoggerOptions
}

func NewCustomLogger(props *CustomLoggerOptions) Logger {
	if props == nil {
		props = &CustomLoggerOptions{}
	}
	if props.InfofFn == nil {
		props.InfofFn = log.Printf
	}
	if props.WarnfFn == nil {
		props.WarnfFn = log.Printf
	}
	if props.PanicfFn == nil {
		props.PanicfFn = log.Panicf
	}
	return &customLogger{opts: props}
}

func (l *customLogger) Infof(msg string, args ...any) {
	l.opts.InfofFn(msg, args...)
}

func (l *customLogger) Warnf(msg string, args ...any) {
	l.opts.WarnfFn(msg, args...)
}

func (l *customLogger) Panicf(msg string, args ...any) {
	l.opts.PanicfFn(msg, args...)
}
