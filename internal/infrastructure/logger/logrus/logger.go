package logrus

import (
	"context"
	"net/http"
	"os"
	
	"telegram-vpn-bot/internal/infrastructure/logger/interfaces"
	
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	internalLogger *logrus.Logger
}

func New() interfaces.Logger {
	log := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	
	log.SetReportCaller(true)
	log.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		HideKeys:        true,
		FieldsOrder:     []string{"component", "category"},
	})
	
	result := &Logger{internalLogger: log}
	
	return result
}

func (l *Logger) Error(args ...interface{}) {
	l.internalLogger.Error(args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.internalLogger.Warn(args...)
}

func (l *Logger) Info(args ...interface{}) {
	l.internalLogger.Info(args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.internalLogger.Debug(args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.internalLogger.Fatal(args...)
}

func (l *Logger) WithField(k string, v interface{}) *interfaces.LoggerEntry {
	return &interfaces.LoggerEntry{
		Fields: map[string]interface{}{
			k: v,
		},
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *interfaces.LoggerEntry {
	return &interfaces.LoggerEntry{
		Fields: fields,
	}
}

func (l *Logger) WithError(err error) *interfaces.LoggerEntry {
	return l.WithField("error", err)
}

func (l *Logger) WithRequest(request *http.Request) *interfaces.LoggerEntry {
	return l.WithField("request", request)
}

func (l *Logger) WithContext(ctx context.Context) *interfaces.LoggerEntry {
	return &interfaces.LoggerEntry{
		Context: ctx,
	}
}
