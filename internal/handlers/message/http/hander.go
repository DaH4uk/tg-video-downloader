package http

import (
	"context"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/infrastructure/metrics"
	"tg-video-downloader/internal/services/messages_sender"
	"tg-video-downloader/internal/services/video_manager"
)

const botDownloadTimeout = 10 * time.Minute

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

	dir, err := os.MkdirTemp("", "tgvd-bot-*")
	if err != nil {
		h.log.WithError(err).Warn("failed to create temp dir")
		_, err = h.messageSender.ReplyTo(message, "failed to download video: "+err.Error(), false)
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			h.log.WithError(err).Warn("failed to clean up temp dir")
		}
	}()

	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), botDownloadTimeout)
	defer downloadCancel()

	videoPath, err := h.videoDownloader.DownloadVideoTo(downloadCtx, message.Text, dir)
	if err != nil {
		h.log.WithError(err).Warn("failed to download video")
		_, err = h.messageSender.ReplyTo(message, "failed to download video: "+err.Error(), false)
		return err
	}

	err = h.messageSender.EditMessage(message.Chat.ID, msg.MessageID, "Uploading video...")
	if err != nil {
		return err
	}

	start := time.Now()
	err = h.messageSender.VideoReplyTo(message, videoPath)
	metrics.UploadDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.UploadTotal.WithLabelValues("error").Inc()
		return err
	}
	metrics.UploadTotal.WithLabelValues("success").Inc()
	return nil
}
