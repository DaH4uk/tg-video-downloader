package start

import (
	"telegram-vpn-bot/internal/services/notifier_service"
	"telegram-vpn-bot/internal/services/users_service"
	
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

type StartHandler struct {
	bot             *tgbotapi.BotAPI
	notifierService *notifier_service.Notifier
	usersService    *users_service.Service
}

func NewStartHandler(
	bot *tgbotapi.BotAPI,
	notifierService *notifier_service.Notifier,
	usersService *users_service.Service,
) *StartHandler {
	return &StartHandler{
		bot:             bot,
		notifierService: notifierService,
		usersService:    usersService,
	}
}

func (h *StartHandler) HandleCommand(message *tgbotapi.Message) error {
	sender := message.From
	
	user := users_service.User{
		TelegramID:       sender.ID,
		FirstName:        sender.FirstName,
		LastName:         sender.LastName,
		TelegramUserName: sender.UserName,
		Confirmed:        false,
		Admin:            false,
	}
	createdUser, err := h.usersService.CreateUser(user)
	if err != nil {
		if errors.Is(err, users_service.UserAlreadyExists) {
			newMessage := tgbotapi.NewMessage(message.Chat.ID, "Ваш запрос на добавление уже был отправлен на обработку, пожалуйста ожидайте")
			
			if createdUser != nil && createdUser.Confirmed {
				newMessage.Text = "Запрос на добавление от вас уже был был подтвержден"
			}
			
			_, err = h.bot.Send(newMessage)
			if err != nil {
				return err
			}
		}
		return err
	}
	
	err = h.notifierService.NotifyAdmins(createdUser)
	if err != nil {
		return err
	}
	
	newMessage := tgbotapi.NewMessage(message.Chat.ID, "Привет! Чтобы продолжить, ваш запрос на добавление должен быть подтвержден администратором, пожалуйста, ожидайте")
	_, err = h.bot.Send(newMessage)
	if err != nil {
		return err
	}
	return nil
}
