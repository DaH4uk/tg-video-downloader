package logrus

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
)

const (
	errorKey               = "error"
	requestKey             = "request"
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 4
)

var (
	// Used for caller information initialisation
	callerInitOnce sync.Once

	minimumCallerDepth = 1

	logrusPackage string
)

type Logger struct {
	logger *logrus.Logger
}

func New() interfaces.Logger {
	log := &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}

	log.SetReportCaller(false)
	log.SetFormatter(&nested.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		HideKeys:        false,
		FieldsOrder:     []string{"component", "category"},
	})

	result := &Logger{logger: log}

	return result
}

func (l Logger) getLoggerEntry() *Entry {
	caller := getCaller()
	return &Entry{
		loggerEntry: l.logger.
			WithField("file", fmt.Sprintf("%s:%v", caller.File, caller.Line)),
	}
}

func (l Logger) Error(args ...interface{}) {
	l.getLoggerEntry().Error(args...)
}

func (l Logger) Warn(args ...interface{}) {
	l.getLoggerEntry().Warn(args...)
}

func (l Logger) Info(args ...interface{}) {
	l.getLoggerEntry().Info(args...)
}

func (l Logger) Debug(args ...interface{}) {
	l.getLoggerEntry().Debug(args...)
}

func (l Logger) Debugf(s string, args ...interface{}) {
	l.getLoggerEntry().Debugf(s, args...)
}

func (l Logger) Fatal(args ...interface{}) {
	l.getLoggerEntry().Error(args...)
}

func (l Logger) WithField(k string, v interface{}) interfaces.Entry {
	return l.getLoggerEntry().WithField(k, v)
}

func (l Logger) WithFields(fields map[string]interface{}) interfaces.Entry {
	return l.getLoggerEntry().WithFields(fields)
}

func (l Logger) WithError(err error) interfaces.Entry {
	return l.WithField(errorKey, err)
}

func (l Logger) WithRequest(request *http.Request) interfaces.Entry {
	return l.WithField(requestKey, request)
}

func (l Logger) WithContext(ctx context.Context) interfaces.Entry {
	return l.WithField("context", ctx)
}

// getCaller retrieves the name of the first non-logrus calling function
func getCaller() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}
