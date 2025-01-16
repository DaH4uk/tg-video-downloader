package messages_sender

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Sender struct {
	bot *tgbotapi.BotAPI
}

func NewMessagesSender(bot *tgbotapi.BotAPI) *Sender {
	return &Sender{
		bot: bot,
	}
}
