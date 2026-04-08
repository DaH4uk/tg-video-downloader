# tg-video-downloader

Telegram bot that accepts `https://` URLs and downloads videos via [yt-dlp](https://github.com/yt-dlp/yt-dlp), then re-uploads them directly to the chat. Stateless, no database.

## How it works

Send the bot any `https://` URL — it will download the video and send it back as a Telegram file. After sending, the local file is deleted.

Supported sources: anything yt-dlp supports (YouTube, Twitter/X, Instagram, TikTok, Vimeo, etc.).

## Requirements

- Go 1.21+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) (installed automatically on first run via go-ytdlp)

## Environment variables

| Variable | Description |
|---|---|
| `TELEGRAM_BOT_TOKEN` | Bot token from [@BotFather](https://t.me/BotFather) |
| `METRICS_USERNAME` | Basic auth username for `/metrics` endpoint |
| `METRICS_PASSWORD` | Basic auth password for `/metrics` endpoint |

## Running locally

```bash
# Copy and fill in the env file
cp .env.example .env

# Run
go run ./cmd/service/main.go
```

## Running with Docker

```bash
docker compose up --build
```

Requires a `.env` file with the variables above.

## Metrics

Prometheus metrics are available at `:9988/metrics` (container port `9900`), protected by HTTP Basic Auth.

## Deployment

On every push to `main`, CI:
1. Lints and builds a Docker image
2. Pushes to `ghcr.io/dah4uk/tg-video-downloader:v0.0.<run_number>`
3. Deploys to the server via SSH (`docker compose pull && up -d`)