package notifier_service

import (
	"fmt"

	"telegram-vpn-bot/internal/infrastructure/logger"
	"telegram-vpn-bot/internal/services/users_service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"gopkg.in/telebot.v4"
)

var (
	log = logger.GetLogger()

	CantGetAdminsError = errors.New("Cannot get admins")

	selector = &telebot.ReplyMarkup{}

	ConfirmUserButton = selector.Text("+ Добавить пользователя")
	CancelUserButton  = selector.Text("× Не добавлять пользователя")
)

type Notifier struct {
	telegramBot *tgbotapi.BotAPI
	userService *users_service.Service
}

func New(telegramBot *tgbotapi.BotAPI, userService *users_service.Service) *Notifier {
	selector.Inline(
		selector.Row(ConfirmUserButton, CancelUserButton),
	)

	return &Notifier{
		telegramBot: telegramBot,
		userService: userService,
	}
}

func (n *Notifier) NotifyAdmins(user *users_service.User) error {
	admins, err := n.userService.GetAdmins()
	if err != nil {
		log.WithField("user", user).Warn("Cannot get admins", err)

		return CantGetAdminsError
	}

	for _, admin := range admins {
		textMessage := fmt.Sprintf(`Добавлен новый пользователь:

Name: %s %s
Username: @%s
`,
			user.FirstName,
			user.LastName,
			user.TelegramUserName,
		)

		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Отклонить", fmt.Sprintf("cancel_user %v", user.ID)),
				tgbotapi.NewInlineKeyboardButtonData("✅ Подтвердить", fmt.Sprintf("confirm_user %v", user.ID)),
			),
		)

		message := tgbotapi.NewMessage(admin.TelegramID, textMessage)
		message.ReplyMarkup = inlineKeyboard

		_, err := n.telegramBot.Send(message)
		if err != nil {
			log.
				WithFields(map[string]interface{}{
					"user_id":     admin.ID,
					"telegram_id": admin.TelegramID,
				}).
				WithError(err).
				Warn("Cannot send message to admin")
		}
	}

	return nil
}
