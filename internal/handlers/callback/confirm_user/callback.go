package confirm_user

import (
	"errors"
	"fmt"
	"strconv"

	"telegram-vpn-bot/internal/infrastructure/logger"
	"telegram-vpn-bot/internal/services/users_service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	log = logger.GetLogger()

	WrongArgumentsError = errors.New("wrong arguments")
)

type Callback struct {
	bot          *tgbotapi.BotAPI
	usersService *users_service.Service
}

func New(
	bot *tgbotapi.BotAPI,
	usersService *users_service.Service,
) *Callback {
	return &Callback{
		bot:          bot,
		usersService: usersService,
	}
}

func (c *Callback) Handle(query *tgbotapi.CallbackQuery, args []string) error {
	if len(args) != 1 {
		return WrongArgumentsError
	}

	argument := args[0]
	id, err := strconv.ParseUint(argument, 10, 32)
	if err != nil {
		return WrongArgumentsError
	}

	user, err := c.usersService.ConfirmUser(uint(id))
	if err != nil {
		if errors.Is(err, users_service.UserAlreadyConfirmed) {
			callback := tgbotapi.NewCallback(query.ID, fmt.Sprintf("🤔 Запрос пользователя @%s уже подтвержден ранее", user.TelegramUserName))
			if _, err = c.bot.Request(callback); err != nil {
				return err
			}
			return nil
		}

		return err
	}

	callback := tgbotapi.NewCallback(query.ID, fmt.Sprintf("✅ Пользователь @%s подтвержден", user.TelegramUserName))
	if _, err = c.bot.Request(callback); err != nil {
		return err
	}

	message := tgbotapi.NewMessage(user.TelegramID, "✅ Ваша заявка принята")
	if _, err := c.bot.Send(message); err != nil {
		log.
			WithField("user", user).
			Warn("sending message error: ", err)
	}

	return nil
}
