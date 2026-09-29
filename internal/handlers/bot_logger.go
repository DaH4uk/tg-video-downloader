package handlers

import (
	"fmt"
	"strings"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
)

// botLogger routes telegram-bot-api's internal logging through the project logger,
// whose formatter redacts the bot token. The library's default logger writes raw
// net/http errors (which embed the token in the request URL) straight to stderr.
type botLogger struct {
	log interfaces.Logger
}

// Println is used by the library for polling errors in GetUpdatesChan.
func (l botLogger) Println(v ...interface{}) {
	l.log.WithField("component", "tgbotapi").Warn(strings.TrimSuffix(fmt.Sprintln(v...), "\n"))
}

// Printf is used by the library for debug request/response dumps.
func (l botLogger) Printf(format string, v ...interface{}) {
	l.log.WithField("component", "tgbotapi").Debug(strings.TrimSuffix(fmt.Sprintf(format, v...), "\n"))
}
