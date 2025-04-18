package messages_sender

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Sender struct {
	bot *tgbotapi.BotAPI
}

func New(bot *tgbotapi.BotAPI) *Sender {
	return &Sender{
		bot: bot,
	}
}

func (s Sender) ReplyTo(message *tgbotapi.Message, text string) (*tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	msg.ReplyToMessageID = message.MessageID

	result, err := s.bot.Send(msg)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s Sender) EditMessage(chatID int64, messageID int, newText string) error {
	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, newText)
	_, err := s.bot.Send(editMsg)
	if err != nil {
		return err
	}

	return nil
}

func (s Sender) DeleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	_, err := s.bot.Send(deleteMsg)
	if err != nil {
		return err
	}

	return nil
}

func (s Sender) VideoReplyTo(message *tgbotapi.Message, videoFilePath string) error {
	file := tgbotapi.FilePath(videoFilePath)
	msg := tgbotapi.NewVideo(message.Chat.ID, file)
	msg.ReplyToMessageID = message.MessageID

	_, err := s.bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}
