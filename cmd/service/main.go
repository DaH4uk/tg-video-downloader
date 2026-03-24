package main

import (
	"context"
	netHttp "net/http"
	"os/signal"
	"syscall"
	"time"

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

	if err := godotenv.Load(); err != nil {
		log.Warn(errors.Wrap(err, "failed to load .env file, using environment variables"))
	}

	telegramBotApi, err := handlers.InitBotApi()
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to init telegram bot api"))
	}

	videoDownloader, err := video_manager.New(log)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to initialize video downloader"))
	}

	messageSender := messages_sender.New(telegramBotApi)
	handler := message_handler.New(telegramBotApi)

	httpMessageHandler := http.New(log, messageSender, videoDownloader)
	handler.RegisterMessageHandler("https://", httpMessageHandler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go handler.HandleUpdates(ctx)

	netHttp.Handle("/metrics", promhttp.Handler())
	srv := &netHttp.Server{Addr: ":8080"}

	go func() {
		log.Info("Metrics server is running on port 8080")
		if err := srv.ListenAndServe(); err != nil && err != netHttp.ErrServerClosed {
			log.WithError(err).Error("metrics server error")
		}
	}()

	<-ctx.Done()
	log.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Warn("HTTP server shutdown error")
	}
}