package main

import (
	"context"
	netHttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"crypto/subtle"

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

func metricsBasicAuth(username, password string, next netHttp.Handler) netHttp.Handler {
	return netHttp.HandlerFunc(func(w netHttp.ResponseWriter, r *netHttp.Request) {
		u, p, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(u), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(p), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
			netHttp.Error(w, "Unauthorized", netHttp.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log := logger.GetLogger()

	if err := godotenv.Load(); err != nil {
		log.Warn(errors.Wrap(err, "failed to load .env file, using environment variables"))
	}

	metricsUsername := os.Getenv("METRICS_USERNAME")
	metricsPassword := os.Getenv("METRICS_PASSWORD")
	if metricsUsername == "" || metricsPassword == "" {
		log.Fatal("METRICS_USERNAME and METRICS_PASSWORD environment variables must be set")
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

	mux := netHttp.NewServeMux()
	mux.Handle("/metrics", metricsBasicAuth(metricsUsername, metricsPassword, promhttp.Handler()))
	srv := &netHttp.Server{Addr: ":9900", Handler: mux}

	go func() {
		log.Info("Metrics server is running on port 9900")
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
