package main

import (
	"fmt"
	"log/slog"
	"os"
)

type Logger struct {
	embeddedLogger *slog.Logger
}

func NewLoggerWithHandler(handler slog.Handler) *Logger {
	return &Logger{embeddedLogger: slog.New(handler)}
}

func NewLogger() *Logger {
	handler := slog.NewTextHandler(os.Stdout, nil)
	return &Logger{embeddedLogger: slog.New(handler)}
}

func (l *Logger) GetSlogger() *slog.Logger {
	return l.embeddedLogger
}

func (l *Logger) Info(v ...interface{}) {
	l.embeddedLogger.Info(fmt.Sprintf("%v", v...))
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.embeddedLogger.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Warn(v ...interface{}) {
	l.embeddedLogger.Warn(fmt.Sprintf("%v", v...))
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.embeddedLogger.Warn(fmt.Sprintf(format, v...))
}

func (l *Logger) Error(v ...interface{}) {
	l.embeddedLogger.Error(fmt.Sprintf("%v", v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.embeddedLogger.Error(fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.embeddedLogger.Error(fmt.Sprintf("%v", v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.embeddedLogger.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) FatalWithCode(code int, v ...interface{}) {
	l.embeddedLogger.Error(fmt.Sprintf("%v", v...))
	os.Exit(int(code))
}
