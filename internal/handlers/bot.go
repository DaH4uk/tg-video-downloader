package handlers

import (
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"telegram-vpn-bot/internal/infrastructure/logger"
)

var log = logger.GetLogger()

func InitBotApi() (*tgbotapi.BotAPI, error) {
	telegramBotToken := os.Getenv("TELEGRAM_BOT_TOKEN")

	if telegramBotToken == "" {
		return nil, errors.New("TELEGRAM_BOT_TOKEN environment variable not set")
	}

	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create telegram bot")
	}

	if os.Getenv("TELEGRAM_BOT_DEBUG") == "true" {
		bot.Debug = true
		log.Info("Telegram bot is in debug mode")
	}

	log.WithField("self_username", bot.Self.UserName).Info("Authorized")

	return bot, nil
}
