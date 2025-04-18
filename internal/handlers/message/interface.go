package message

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Handler interface {
	HandleMessage(message *tgbotapi.Message) error
}
