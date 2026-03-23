package logger

import (
	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/infrastructure/logger/logrus"
)

var (
	logger = logrus.New()
)

func GetLogger() interfaces.Logger {
	return logger
}
