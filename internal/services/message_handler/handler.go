package message_handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"tg-video-downloader/internal/handlers/message"
	"tg-video-downloader/internal/infrastructure/logger"
	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/infrastructure/metrics"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const maxConcurrent = 5

type TelegramHandler struct {
	bot             *tgbotapi.BotAPI
	messageHandlers map[string]message.Handler
	log             interfaces.Logger
	sem             chan struct{}
}

func New(bot *tgbotapi.BotAPI) *TelegramHandler {
	return &TelegramHandler{
		bot:             bot,
		messageHandlers: map[string]message.Handler{},
		log:             logger.GetLogger(),
		sem:             make(chan struct{}, maxConcurrent),
	}
}

func (h *TelegramHandler) RegisterMessageHandler(messagePrefix string, handler message.Handler) {
	h.messageHandlers[strings.ToLower(messagePrefix)] = handler
}

func (h *TelegramHandler) HandleUpdates(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := h.bot.GetUpdatesChan(u)
	h.log.Info("Handling telegram requests")

	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			h.bot.StopReceivingUpdates()
			wg.Wait()
			return
		case update, ok := <-updates:
			if !ok {
				wg.Wait()
				return
			}
			wg.Add(1)
			go h.handleUpdate(&wg, update)
		}
	}
}

func (h *TelegramHandler) handleUpdate(wg *sync.WaitGroup, update tgbotapi.Update) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			metrics.MessagesProcessed.WithLabelValues("error").Inc()
			h.log.WithField("panic", fmt.Sprintf("%v", r)).Error("recovered from panic in update handler")
		}
	}()

	h.sem <- struct{}{}
	defer func() { <-h.sem }()

	isTextMessage := update.Message != nil && update.Message.Text != ""
	if isTextMessage {
		metrics.MessagesReceived.Inc()
	}

	if err := h.handleMessage(update.Message); err != nil {
		metrics.MessagesProcessed.WithLabelValues("error").Inc()
		h.log.Warn("handle message error: ", err)
	} else if isTextMessage {
		metrics.MessagesProcessed.WithLabelValues("success").Inc()
	}
}

func (h *TelegramHandler) handleMessage(message *tgbotapi.Message) error {
	if message == nil || message.Text == "" {
		return nil
	}

	h.log.Debugf("[%s] %s", message.From.UserName, message.Text)

	messageText := strings.ToLower(message.Text)
	for messagePrefix, handler := range h.messageHandlers {
		if strings.HasPrefix(messageText, messagePrefix) {
			return handler.HandleMessage(message)
		}
	}

	h.log.Debugf("Message handler not found for message: %s", message.Text)
	return nil
}
