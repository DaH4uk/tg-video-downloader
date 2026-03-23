package http

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/services/messages_sender"
	"tg-video-downloader/internal/services/video_manager"
)

type MessageHandler struct {
	messageSender   *messages_sender.Sender
	videoDownloader video_manager.VideoManager
	log             interfaces.Logger
}

func New(log interfaces.Logger, messageSender *messages_sender.Sender, videoDownloader video_manager.VideoManager) *MessageHandler {
	return &MessageHandler{
		log:             log,
		messageSender:   messageSender,
		videoDownloader: videoDownloader,
	}
}

func (h MessageHandler) HandleMessage(message *tgbotapi.Message) error {
	if !strings.HasPrefix(message.Text, "https://") {
		h.log.WithField("message", message).Warn("invalid URL format: must start with https://")
		_, err := h.messageSender.ReplyTo(message, "invalid URL format: must start with https://", false)
		return err
	}

	msg, err := h.messageSender.ReplyTo(message, "Downloading video...", true)
	if err != nil {
		return err
	}
	defer func(messageSender *messages_sender.Sender, chatID int64, messageID int) {
		_ = messageSender.DeleteMessage(chatID, messageID)
	}(h.messageSender, message.Chat.ID, msg.MessageID)

	videoPath, err := h.videoDownloader.DownloadVideo(message.Text)

	defer func(videoDownloader video_manager.VideoManager, fileName string) {
		err := videoDownloader.DeleteVideo(fileName)
		h.log.Info("deleted video:" + fileName)
		if err != nil {
			h.log.WithError(err).Warn("failed to delete video")
		}
	}(h.videoDownloader, videoPath)
	if err != nil {
		h.log.WithError(err).Warn("failed to download video")
		_, err = h.messageSender.ReplyTo(message, "failed to download video: "+err.Error(), false)
		return err
	}

	err = h.messageSender.EditMessage(message.Chat.ID, msg.MessageID, "Uploading video...")
	if err != nil {
		return err
	}
	return h.messageSender.VideoReplyTo(message, videoPath)
}
