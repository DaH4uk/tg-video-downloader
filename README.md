# tg-video-downloader

Telegram bot that accepts `https://` URLs and downloads videos via [yt-dlp](https://github.com/yt-dlp/yt-dlp), then re-uploads them directly to the chat. Stateless, no database.

It also exposes an HTTP endpoint that returns the downloaded file, so an Apple Shortcut can save videos straight to the camera roll.

## How it works

Send the bot any `https://` URL — it will download the video and send it back as a Telegram file. After sending, the local file is deleted.

Supported sources: anything yt-dlp supports (YouTube, Twitter/X, Instagram, TikTok, Vimeo, etc.).

Videos larger than 500 MB or longer than 15 minutes are rejected — both the bot and the HTTP API share these limits.

## Requirements

- Go 1.26+
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) (installed automatically on first run via go-ytdlp)
- `ffmpeg` and `ffprobe` — used by the HTTP API to convert videos Apple Photos would otherwise reject. The service refuses to start without them.

## Environment variables

| Variable | Description |
|---|---|
| `TELEGRAM_BOT_TOKEN` | Bot token from [@BotFather](https://t.me/BotFather) |
| `METRICS_USERNAME` | Basic auth username for `/metrics` endpoint |
| `METRICS_PASSWORD` | Basic auth password for `/metrics` endpoint |
| `API_TOKEN` | Bearer token for `POST /api/download`; the endpoint is disabled when unset |

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

## HTTP download API

`POST /api/download` downloads a video and returns the file itself. Disabled unless `API_TOKEN` is set — the bot keeps working either way.

```bash
curl -X POST https://<host>/api/download \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.tiktok.com/@user/video/123"}' \
  -o video.mp4
```

The response is `video/mp4` with a `Content-Disposition` filename. Errors come back as `{"error": "..."}`:

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

Each request downloads into its own temp directory, which is removed once the response is sent. Unlike the bot path, the API converts the file to H.264/AAC when it is not already in that format — Apple Photos rejects anything else. Files that already qualify (most TikTok, Reels and Shorts) skip ffmpeg entirely.

## Apple Shortcuts

1. New shortcut → **Get Contents of URL**
   - URL: `https://<host>/api/download`
   - Method: `POST`
   - Headers: `Authorization` = `Bearer <your token>`
   - Request Body: `JSON`, one field `url` (text) = **Shortcut Input**
2. Add **Save to Photo Album**.
3. In the shortcut settings, enable **Show in Share Sheet** and set the accepted input type to **URLs**.

Sharing a link from TikTok, Instagram or YouTube now saves the video straight to the camera roll. Long videos are rejected by design — the endpoint targets short clips.

## Metrics

Prometheus metrics are available at `:9988/metrics` (container port `9900`), protected by HTTP Basic Auth. The download API runs on the same port.

## Deployment

On every push to `main`, CI:
1. Lints, builds and tests, then builds a Docker image
2. Pushes to `ghcr.io/dah4uk/tg-video-downloader:v0.0.<run_number>`
3. Deploys to the server via SSH (`docker compose pull && up -d`)

The deploy step writes `API_TOKEN` from the repository secret of the same name; without that secret the endpoint stays disabled.
