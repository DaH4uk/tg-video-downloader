package interfaces

import (
	"context"
	"net/http"
)

type EntryObject struct {
	Context context.Context
}

type Entry interface {
	WithField(k string, v interface{}) Entry
	WithFields(fields map[string]interface{}) Entry
	WithError(err error) Entry
	WithRequest(request *http.Request) Entry
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Debug(args ...interface{})
	Debugf(s string, args ...interface{})
}
