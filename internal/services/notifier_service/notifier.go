package notifier

import (
	"telegram_vpn_bot/internal/services/users_service"
	
	"gopkg.in/telebot.v4"
)

type Notifier struct {
	telegramBot *telebot.Bot
	userService *users_service.Service
}

func NewNotifier(userService *users_service.Service) *Notifier {
	return &Notifier{
		userService: userService,
	}
}

func (n *Notifier) NotifyAdmins() {
	// TODO: implement
}
