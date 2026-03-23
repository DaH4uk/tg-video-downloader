package main

import (
	netHttp "net/http"

	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"tg-video-downloader/internal/handlers"
	"tg-video-downloader/internal/handlers/message/http"
	"tg-video-downloader/internal/infrastructure/logger"
	"tg-video-downloader/internal/services/message_handler"
	"tg-video-downloader/internal/services/messages_sender"
	"tg-video-downloader/internal/services/video_manager"
)

func main() {
	log := logger.GetLogger()
	err := godotenv.Overload()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to load .env file"))
	}

	telegramBotApi, err := handlers.InitBotApi()
	if err != nil {
		panic(errors.Wrap(err, "failed to init telegram bot api"))
	}

	videoDownloader := video_manager.New(log)
	messageSender := messages_sender.New(telegramBotApi)

	handler := message_handler.New(telegramBotApi)

	httpMessageHandler := http.New(log, messageSender, videoDownloader)
	handler.RegisterMessageHandler("https://", httpMessageHandler)

	go handler.HandleUpdates()

	netHttp.Handle("/metrics", promhttp.Handler())
	log.Info("Metrics server is running on port 8080")
	log.Fatal(netHttp.ListenAndServe(":8080", nil))
}
