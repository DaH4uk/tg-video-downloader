package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "tgvd"

var (
	// MessagesReceived counts all valid text messages received by the bot.
	MessagesReceived = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "messages_received_total",
		Help:      "Total number of text messages received by the bot",
	})

	// MessagesProcessed counts messages that were routed to a handler, by status.
	MessagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "messages_processed_total",
		Help:      "Total number of messages processed by the bot",
	}, []string{"status"})

	// DownloadDuration tracks yt-dlp download time in seconds.
	DownloadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "download_duration_seconds",
		Help:      "Duration of video download via yt-dlp in seconds",
		Buckets:   []float64{5, 10, 30, 60, 120, 300, 600},
	})

	// DownloadTotal counts download attempts by status.
	DownloadTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "download_total",
		Help:      "Total number of video download attempts",
	}, []string{"status"})

	// UploadDuration tracks Telegram video upload time in seconds.
	UploadDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "upload_duration_seconds",
		Help:      "Duration of video upload to Telegram in seconds",
		Buckets:   []float64{5, 10, 30, 60, 120, 300, 600},
	})

	// UploadTotal counts upload attempts by status.
	UploadTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "upload_total",
		Help:      "Total number of video upload attempts to Telegram",
	}, []string{"status"})
)
