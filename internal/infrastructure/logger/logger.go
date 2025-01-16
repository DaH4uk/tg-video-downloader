package logger

import (
	"telegram-vpn-bot/internal/infrastructure/logger/interfaces"
	"telegram-vpn-bot/internal/infrastructure/logger/logrus"
)

var (
	logger = logrus.New()
)

func GetLogger() interfaces.Logger {
	return logger
}
