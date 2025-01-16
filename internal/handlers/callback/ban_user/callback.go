package cancel

import (
	"telegram-vpn-bot/internal/services/users_service"
	
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type Callback struct {
	bot          *tgbotapi.BotAPI
	logger       *logrus.Logger
	usersService *users_service.Service
}

func New(bot *tgbotapi.BotAPI, logger *logrus.Logger, usersService *users_service.Service) *Callback {
	return &Callback{
		bot:          bot,
		logger:       logger,
		usersService: usersService,
	}
}

func (c *Callback) Handle(query *tgbotapi.CallbackQuery, args []string) error {

}
