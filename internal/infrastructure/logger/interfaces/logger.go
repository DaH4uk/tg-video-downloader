package interfaces

import (
	"context"
	"net/http"
)

type Logger interface {
	Error(args ...interface{})
	Warn(args ...interface{})
	Info(args ...interface{})
	Debug(args ...interface{})
	Debugf(s string, args ...interface{})
	Fatal(args ...interface{})

	WithField(k string, v interface{}) Entry
	WithFields(fields map[string]interface{}) Entry
	WithError(err error) Entry
	WithRequest(request *http.Request) Entry
	WithContext(ctx context.Context) Entry
}
