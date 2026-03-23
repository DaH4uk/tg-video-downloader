package logrus

import (
	"net/http"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"

	"github.com/sirupsen/logrus"
)

type Entry struct {
	interfaces.EntryObject
	loggerEntry *logrus.Entry
}

func (e Entry) WithField(k string, v interface{}) interfaces.Entry {
	return &Entry{
		EntryObject: e.EntryObject,
		loggerEntry: e.loggerEntry.WithField(k, v),
	}
}

func (e Entry) WithFields(fields map[string]interface{}) interfaces.Entry {
	return Entry{
		EntryObject: e.EntryObject,
		loggerEntry: e.loggerEntry.WithFields(fields),
	}
}

func (e Entry) WithError(err error) interfaces.Entry {
	return Entry{
		EntryObject: e.EntryObject,
		loggerEntry: e.loggerEntry.WithError(err),
	}
}

func (e Entry) WithRequest(request *http.Request) interfaces.Entry {
	return Entry{
		EntryObject: e.EntryObject,
		loggerEntry: e.loggerEntry.WithField(requestKey, request),
	}
}

func (e Entry) Info(args ...interface{}) {
	e.loggerEntry.Info(args...)
}

func (e Entry) Warn(args ...interface{}) {
	e.loggerEntry.Warn(args...)
}

func (e Entry) Error(args ...interface{}) {
	e.loggerEntry.Error(args...)
}

func (e Entry) Debug(args ...interface{}) {
	e.loggerEntry.Debug(args...)
}

func (e Entry) Debugf(s string, args ...interface{}) {
	e.loggerEntry.Debugf(s, args...)
}
