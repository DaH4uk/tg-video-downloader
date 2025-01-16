package interfaces

import (
	"context"
	"net/http"
)

type LoggerEntry struct {
	Fields map[string]interface{}
}

func (e *LoggerEntry) WithField(ctx context.Context, k string, v interface{}) *LoggerEntry {
	e.Fields["ctx"] = ctx
	e.Fields[k] = v
	return e
}

func (e *LoggerEntry) WithFields(ctx context.Context, fields map[string]interface{}) *LoggerEntry {
	e.Fields["ctx"] = ctx
	for k, v := range fields {
		e.Fields[k] = v
	}
	return e
}

func (e *LoggerEntry) WithError(ctx context.Context, err error) *LoggerEntry {
	e.Fields["ctx"] = ctx
	e.Fields["err"] = err
	return e
}

func (e *LoggerEntry) WithRequest(ctx context.Context, request *http.Request) *LoggerEntry {
	e.Fields["ctx"] = ctx
	e.Fields["req"] = request
	return e
}
