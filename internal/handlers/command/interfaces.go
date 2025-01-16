package message

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type CommandHandler interface {
	HandleCommand(message *tgbotapi.Message) error
}
