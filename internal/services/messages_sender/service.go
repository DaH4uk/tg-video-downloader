package messages_sender

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Sender struct {
	bot *tgbotapi.BotAPI
}

func New(bot *tgbotapi.BotAPI) *Sender {
	return &Sender{
		bot: bot,
	}
}

func (s Sender) ReplyTo(message *tgbotapi.Message, text string, silent bool) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID
	msg.DisableNotification = silent

	result, err := s.bot.Send(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	return &result, nil
}

func (s Sender) EditMessage(chatID int64, messageID int, newText string) error {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	_, err := s.bot.Send(editMsg)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

func (s Sender) DeleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := s.bot.Send(deleteMsg)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

func (s Sender) VideoReplyTo(message *tgbotapi.Message, videoFilePath string) error {
	file := tgbotapi.FilePath(videoFilePath)
	msg := tgbotapi.NewVideo(message.Chat.ID, file)
	msg.ReplyToMessageID = message.MessageID

	_, err := s.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("can't send video reply: %w", err)
	}

	return nil
}
