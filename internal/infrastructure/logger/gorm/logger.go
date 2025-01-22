package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"telegram-vpn-bot/internal/infrastructure/logger"
	"telegram-vpn-bot/internal/infrastructure/logger/interfaces"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

const sqlMessageTemplate = "%s [%s]"

type Logger struct {
	SlowThreshold         time.Duration
	SourceField           string
	SkipErrRecordNotFound bool
	Debug                 bool
	log                   interfaces.Logger
}

func New() *Logger {
	return &Logger{
		SkipErrRecordNotFound: true,
		Debug:                 true,
		log:                   logger.GetLogger(),
	}
}

func (l *Logger) LogMode(gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *Logger) Info(ctx context.Context, s string, args ...interface{}) {
	l.log.
		WithContext(ctx).
		Info(fmt.Sprint(s, args))
}

func (l *Logger) Warn(ctx context.Context, s string, args ...interface{}) {
	l.log.
		WithContext(ctx).
		Warn(fmt.Sprintf(s, args...))
}

func (l *Logger) Error(ctx context.Context, s string, args ...interface{}) {
	l.log.
		WithContext(ctx).
		Error(fmt.Sprintf(s, args...))
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, _ := fc()
	fields := map[string]interface{}{}
	if l.SourceField != "" {
		fields[l.SourceField] = utils.FileWithLineNum()
	}
	if err != nil && !(errors.Is(err, gorm.ErrRecordNotFound) && l.SkipErrRecordNotFound) {
		l.log.
			WithContext(ctx).
			WithError(err).
			Error(fmt.Sprintf(sqlMessageTemplate, sql, elapsed))
		return
	}

	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold {
		l.log.
			WithContext(ctx).
			WithFields(fields).
			Warn(fmt.Sprintf(sqlMessageTemplate, sql, elapsed))
		return
	}

	if l.Debug {
		l.log.
			WithContext(ctx).
			WithFields(fields).
			Debugf(fmt.Sprintf(sqlMessageTemplate, sql, elapsed))
	}
}
