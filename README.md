# tg-video-downloader

Telegram bot that accepts `https://` URLs and downloads/re-uploads videos via yt-dlp. No database. Stateless.

Videos over 500 MB or longer than 15 minutes are rejected (yt-dlp `--max-filesize` / `--match-filter`). This applies to both the Telegram bot and the HTTP API below — it's a deliberate guard, not a bug. If the bot replies "failed to download video: video does not pass the duration filter" or "... is larger than the allowed size", that's why.

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

**Request flow (Telegram):** Telegram update → `message_handler` routes by prefix → `http.MessageHandler` downloads via yt-dlp → sends video file back to chat → deletes local file.

Only one message handler is registered: `"https://"` prefix → `http.MessageHandler`.

**No user authorization** — the bot responds to any Telegram user.

## HTTP download API

`POST /api/download` downloads a video and returns the file itself. Disabled unless `API_TOKEN` is set.

```bash
curl -X POST https://<host>/api/download \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.tiktok.com/@user/video/123"}' \
  -o video.mp4
```

The response is `video/mp4` with a `Content-Disposition` filename. Errors come back as `{"error": "..."}` with:

| Status | Meaning |
|---|---|
| 400 | malformed body, non-https URL, or a private/loopback host |
| 401 | missing or wrong token |
| 405 | request method is not POST |
| 413 | video is over 500 MB or longer than 15 minutes |
| 429 | two downloads are already running |
| 500 | failed to create temp directory, open file, or stat video |
| 502 | yt-dlp or ffmpeg failed |
| 504 | the download did not finish within 5 minutes |

Videos are downloaded into a per-request temp directory and deleted once the response is sent. Transcoding to H.264/AAC happens only when the source file is not already in that format.

## Apple Shortcuts

1. New shortcut → **Get Contents of URL**
   - URL: `https://<host>/api/download`
   - Method: `POST`
   - Headers: `Authorization` = `Bearer <your token>`
   - Request Body: `JSON`, one field `url` (text) = **Shortcut Input**
2. Add **Save to Photo Album**.
3. Open shortcut settings, enable **Show in Share Sheet**, and set the accepted input type to **URLs**.

Sharing a link from TikTok, Instagram or YouTube now saves the video straight to the camera roll. Long videos are rejected by design — the endpoint targets short clips.

## Deployment

CI (`.github/workflows/main.yml`) on push to `main`:
1. Lint → build Docker image → push to `ghcr.io/dah4uk/tg-video-downloader:v0.0.<run_number>`
2. SCP `docker-compose.deploy.yml` to server, SSH deploy via `docker compose pull && up -d`

Local Docker:
```bash
docker compose up --build
```

## Metrics

Metrics endpoint: `:9988/metrics` (mapped from container port 9900 in docker-compose), protected by HTTP Basic Auth. The HTTP API endpoint (`POST /api/download`) runs on the same port and is enabled only when `API_TOKEN` is set.
