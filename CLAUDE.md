# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Telegram bot that accepts `https://` URLs and downloads/re-uploads videos via yt-dlp. No database. Stateless.

## Commands

```bash
# Build
go build ./...

# Run locally (requires .env)
go run ./cmd/service/main.go

# Lint (must pass before merge)
gofmt -l .
golangci-lint run

# Format
gofmt -w .
```

No tests exist in the codebase.

## Required environment variables

| Variable | Description |
|---|---|
| `TELEGRAM_BOT_TOKEN` | Bot token from @BotFather |
| `METRICS_USERNAME` | Basic auth username for `/metrics` |
| `METRICS_PASSWORD` | Basic auth password for `/metrics` |

## Architecture

```
cmd/service/main.go          — wiring: init bot, ytdlp, handlers; runs metrics HTTP server
internal/handlers/
  bot.go                     — initializes tgbotapi.BotAPI from env
  message/
    interface.go             — Handler interface
    http/hander.go           — handles URL messages: download → upload → cleanup
internal/services/
  message_handler/handler.go — routes incoming updates to registered handlers by message prefix
  messages_sender/service.go — thin wrapper around tgbotapi send/edit/delete
  video_manager/service.go   — wraps go-ytdlp: install, download, delete
internal/infrastructure/
  logger/                    — logrus-based logger with interface
  metrics/metrics.go         — Prometheus counters/histograms (namespace: tgvd)
```

**Request flow:** Telegram update → `message_handler` routes by prefix → `http.MessageHandler` downloads via yt-dlp → sends video file back to chat → deletes local file.

Only one message handler is registered: `"https://"` prefix → `http.MessageHandler`.

**No user authorization** — the bot responds to any Telegram user.

## Deployment

CI (`.github/workflows/main.yml`) on push to `main`:
1. Lint → build Docker image → push to `ghcr.io/dah4uk/tg-video-downloader:v0.0.<run_number>`
2. SCP `docker-compose.deploy.yml` to server, SSH deploy via `docker compose pull && up -d`

Local Docker:
```bash
docker compose up --build
```

Metrics endpoint: `:9988/metrics` (mapped from container port 9900), protected by HTTP Basic Auth.