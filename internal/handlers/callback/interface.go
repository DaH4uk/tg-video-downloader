package callback

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type CallbackHandler interface {
	Handle(message *tgbotapi.CallbackQuery, args []string) error
}
