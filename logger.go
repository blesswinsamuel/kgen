package kgen

import "log"

// Logger is an interface for logging
type Logger interface {
	// Infof logs an info message
	Infof(msg string, args ...any)
	// Warnf logs a warning message
	Warnf(msg string, args ...any)
	// Panicf logs a panic message and panics
	Panicf(msg string, args ...any)
}

// CustomLoggerOptions is a struct that contains the options for a custom logger
type CustomLoggerOptions struct {
	// InfofFn is a custom function that logs an info message. If not provided, log.Printf is used.
	InfofFn func(msg string, args ...any)
	// WarnfFn is a custom function that logs a warning message. If not provided, log.Printf is used.
	WarnfFn func(msg string, args ...any)
	// PanicfFn is a custom function that logs a panic message and panics. If not provided, log.Panicf is used.
	PanicfFn func(msg string, args ...any)
}

type customLogger struct {
	opts *CustomLoggerOptions
}

// NewCustomLogger creates a new custom logger.
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
