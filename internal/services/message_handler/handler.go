package message_handler

import (
	"strings"
	"sync"

	"telegram-vpn-bot/internal/handlers/message"
	"telegram-vpn-bot/internal/infrastructure/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

var log = logger.GetLogger()
var wg sync.WaitGroup

type TelegramHandler struct {
	bot             *tgbotapi.BotAPI
	messageHandlers map[string]message.Handler
}

func New(bot *tgbotapi.BotAPI) *TelegramHandler {
	return &TelegramHandler{
		bot:             bot,
		messageHandlers: map[string]message.Handler{},
	}
}

func (h *TelegramHandler) RegisterMessageHandler(messagePrefix string, handler message.Handler) {
	h.messageHandlers[strings.ToLower(messagePrefix)] = handler
}

func (h *TelegramHandler) HandleUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)
	log.Info("Handling telegram requests")
	for update := range updates {
		wg.Add(1)
		go h.handleUpdate(update)
	}
	wg.Wait()
}

func (h *TelegramHandler) handleUpdate(update tgbotapi.Update) {
	defer wg.Done()
	m := update.Message

	err := h.handleMessage(m)
	if err != nil {
		log.Warn("handle m error: ", err)
	}
}

func (h *TelegramHandler) handleMessage(message *tgbotapi.Message) error {
	if message == nil || message.Text == "" {
		return nil
	}

	log.Debugf("[%s] %s", message.From.UserName, message.Text)

	messageText := strings.ToLower(message.Text)
	for messagePrefix, handler := range h.messageHandlers {
		if strings.HasPrefix(messageText, messagePrefix) {
			return handler.HandleMessage(message)
		}
	}

	log.Debugf("Message handler not found for message: %s", message.Text)

	return nil
}

func (h *TelegramHandler) handleEmptyCommand(message *tgbotapi.Message) error {
	messageText := tgbotapi.NewMessage(message.Chat.ID, "Такая команда не найдена")
	_, err := h.bot.Send(messageText)
	if err != nil {
		return errors.Wrap(err, "sending command response error")
	}

	return nil
}
