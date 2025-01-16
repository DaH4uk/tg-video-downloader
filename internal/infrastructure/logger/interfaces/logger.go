package interfaces

import (
	"context"
	"net/http"
)

type Logger interface {
	Error(ctx context.Context, args ...interface{})
	Warning(ctx context.Context, args ...interface{})
	WithField(ctx context.Context, k string, v interface{}) *LoggerEntry
	WithFields(ctx context.Context, fields map[string]interface{}) *LoggerEntry
	WithError(ctx context.Context, err error) *LoggerEntry
	WithRequest(ctx context.Context, request *http.Request) *LoggerEntry
	Info(ctx context.Context, args ...interface{})
	Debug(ctx context.Context, args ...interface{})
}
