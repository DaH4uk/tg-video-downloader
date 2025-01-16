package message_handler

import (
	"strings"

	"telegram-vpn-bot/internal/handlers/callback"
	"telegram-vpn-bot/internal/handlers/command"
	"telegram-vpn-bot/internal/infrastructure/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
)

var log = logger.GetLogger()

type TelegramHandler struct {
	bot              *tgbotapi.BotAPI
	commandHandlers  map[string]command.CommandHandler
	callbackHandlers map[string]callback.CallbackHandler
}

func New(bot *tgbotapi.BotAPI) *TelegramHandler {
	return &TelegramHandler{
		bot:              bot,
		commandHandlers:  map[string]command.CommandHandler{},
		callbackHandlers: map[string]callback.CallbackHandler{},
	}
}

func (h *TelegramHandler) RegisterCommandHandler(commandName string, handler command.CommandHandler) {
	h.commandHandlers[commandName] = handler
}

func (h *TelegramHandler) RegisterCallback(callbackPrefix string, handler callback.CallbackHandler) {
	h.callbackHandlers[strings.ToLower(callbackPrefix)] = handler
}

func (h *TelegramHandler) HandleUpdates() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)
	for update := range updates {
		message := update.Message
		if message == nil {
			h.handleCallback(update)
			continue
		}

		if message.IsCommand() {
			err := h.handleCommand(message)
			if err != nil {
				h.handleCommandError(message, err)

			}
			continue
		}

		err := h.handleMessage(message)
		if err != nil {
			log.Warn("handle message error: ", err)
		}
	}
}

func (h *TelegramHandler) handleCallback(update tgbotapi.Update) {
	callbackQuery := update.CallbackQuery
	if callbackQuery == nil {
		return
	}

	data := callbackQuery.Data
	if data == "" {
		h.handleCallbackError(callbackQuery, errors.New("data is empty"))
		return
	}

	words := strings.Split(data, " ")
	callbackPrefix := strings.ToLower(words[0])

	if handler, ok := h.callbackHandlers[callbackPrefix]; ok {
		var args []string
		if len(words) > 1 {
			args = words[1:]
		}

		err := handler.Handle(callbackQuery, args)
		if err != nil {
			h.handleCallbackError(callbackQuery, err)

			return
		}

		return
	}

	h.handleCallbackError(callbackQuery, errors.New("callback handler not found"))
}

func (h *TelegramHandler) handleCommand(message *tgbotapi.Message) error {
	cmd := message.Command()
	if cmd == "" {
		return nil
	}

	if handler, ok := h.commandHandlers[cmd]; ok {
		log.Debugf("handle command: %s", cmd)

		return handler.HandleCommand(message)
	}

	log.Debugf("unknown command: %s", cmd)
	return h.handleEmptyCommand(message)
}

func (h *TelegramHandler) handleMessage(message *tgbotapi.Message) error {
	if message == nil { // If we got a message
		return nil
	}
	log.Debugf("[%s] %s", message.From.UserName, message.Text)

	msg := tgbotapi.NewMessage(message.Chat.ID, message.Text)
	msg.ReplyToMessageID = message.MessageID

	_, err := h.bot.Send(msg)
	if err != nil {
		return err
	}
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

func (h *TelegramHandler) handleCommandError(message *tgbotapi.Message, err error) {
	messageText := tgbotapi.NewMessage(message.Chat.ID, "Что-то пошло не так: "+err.Error())
	log.
		WithField("chatId", message.Chat.ID).
		Warn("handle command error: ", err)
	_, err = h.bot.Send(messageText)
	if err != nil {
		log.
			WithField("chatId", message.Chat.ID).
			Warn("can't send error message: ", err)
	}
}

func (h *TelegramHandler) handleCallbackError(query *tgbotapi.CallbackQuery, reason error) {
	defaultCallback := tgbotapi.NewCallback(query.ID, "⚠️ Что-то пошло не так")
	log.Warn("handle callback error: ", reason)
	if _, err := h.bot.Request(defaultCallback); err != nil {
		log.Warn("sending callback request error: ", err)
	}
}
