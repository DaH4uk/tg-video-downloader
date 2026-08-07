# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

Telegram bot that accepts `https://` URLs and downloads/re-uploads videos via yt-dlp. No database. Stateless.

Videos over 500 MB or longer than 15 minutes are rejected (yt-dlp `--max-filesize` / `--match-filter`, set in `internal/services/video_manager/service.go`). This applies to both the Telegram bot and the HTTP API — it's a deliberate guard, not a bug.

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

# Test
go test ./...
```

Tests live in `internal/handlers/api` and `internal/services/video_manager`, run via `go test ./...`. Some tests in `internal/services/video_manager/probe_test.go` skip automatically when ffmpeg/ffprobe binaries are absent.

## Required environment variables

| Variable | Description |
|---|---|
| `TELEGRAM_BOT_TOKEN` | Bot token from @BotFather |
| `METRICS_USERNAME` | Basic auth username for `/metrics` |
| `METRICS_PASSWORD` | Basic auth password for `/metrics` |
| `API_TOKEN` | Bearer token for `POST /api/download`; the endpoint is disabled when unset |

## Architecture

```
cmd/service/main.go          — wiring: init bot, ytdlp, handlers; runs HTTP server
internal/handlers/
  bot.go                     — initializes tgbotapi.BotAPI from env
  message/
    interface.go             — Handler interface
    http/hander.go           — handles URL messages: download → upload → cleanup
  api/
    download.go              — POST /api/download: validate → download → probe → transcode → stream file
    auth.go                  — bearer token middleware
    url.go                   — URL validation (https only, no private hosts)
    disposition.go           — RFC 5987 Content-Disposition
internal/services/
  message_handler/handler.go — routes incoming updates to registered handlers by message prefix
  messages_sender/service.go — thin wrapper around tgbotapi send/edit/delete
  video_manager/service.go   — wraps go-ytdlp: install, download, delete, probe, transcode
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

Metrics endpoint: `:9988/metrics` (mapped from container port 9900 in docker-compose), protected by HTTP Basic Auth. The HTTP API endpoint (`POST /api/download`) runs on the same port and is enabled only when `API_TOKEN` is set.