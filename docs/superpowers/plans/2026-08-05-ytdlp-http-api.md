# HTTP-ручка для скачивания видео — план реализации

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Добавить в сервис синхронную ручку `POST /api/download`, которая принимает URL видео и возвращает готовый mp4, пригодный для сохранения в Фотоплёнку из Apple Shortcuts.

**Architecture:** Ручка живёт на существующем HTTP-сервере рядом с `/metrics`. Внутри — знакомый пайплайн `yt-dlp → (ffprobe) → ffmpeg`, вынесенный в методы `video_manager`, которые принимают контекст и целевой каталог. Файлы каждого запроса лежат в своём временном каталоге и удаляются по завершении. Состояния нет: ни очередей, ни хранилища, ни БД.

**Tech Stack:** Go 1.26, stdlib `net/http`, `github.com/lrstanley/go-ytdlp` v1.3.5, `github.com/pkg/errors`, Prometheus client, ffmpeg/ffprobe из образа.

**Spec:** `docs/superpowers/specs/2026-08-05-ytdlp-http-api-design.md`

## Global Constraints

- Go 1.26 (`go.mod`). Новые зависимости не добавлять — только stdlib и то, что уже в `go.mod`.
- Тесты пишутся на стандартном `testing`, без testify и прочих хелперов.
- Комментарии, логи и сообщения об ошибках в коде — на английском, как в существующих файлах.
- Перед каждым коммитом: `gofmt -l .` пусто и `golangci-lint run` без замечаний (конфига в репозитории нет, используются дефолты).
- Лимиты и константы, ровно эти значения: `--max-filesize 500M`, `--match-filter "duration < 900"`, максимум 2 параллельных API-загрузки, дедлайн запроса 5 минут, таймаут загрузки бота 10 минут.
- Ошибки заворачиваются через `github.com/pkg/errors` (`errors.Wrap`), как в существующем коде.
- Ветка: `feature/http-download-api`.

---

### Task 1: Валидация URL запроса

**Files:**
- Create: `internal/handlers/api/url.go`
- Test: `internal/handlers/api/url_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `func ValidateVideoURL(raw string) error`, `var ErrInvalidURL error`, `var ErrForbiddenHost error` в пакете `api`.

- [ ] **Step 1: Написать падающий тест**

Создать `internal/handlers/api/url_test.go`:

```go
package api

import (
	"testing"

	"github.com/pkg/errors"
)

func TestValidateVideoURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want error
	}{
		{"valid https url", "https://www.tiktok.com/@user/video/123", nil},
		{"valid public ip literal", "https://93.184.216.34/video.mp4", nil},
		{"http scheme is rejected", "http://example.com/video", ErrInvalidURL},
		{"missing scheme", "example.com/video", ErrInvalidURL},
		{"empty string", "", ErrInvalidURL},
		{"scheme without host", "https://", ErrInvalidURL},
		{"loopback ipv4", "https://127.0.0.1/video", ErrForbiddenHost},
		{"loopback ipv6", "https://[::1]/video", ErrForbiddenHost},
		{"private network", "https://192.168.1.10/video", ErrForbiddenHost},
		{"link local metadata", "https://169.254.169.254/latest/meta-data", ErrForbiddenHost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateVideoURL(tt.raw)
			if !errors.Is(got, tt.want) {
				t.Fatalf("ValidateVideoURL(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/handlers/api/ -run TestValidateVideoURL -v`
Expected: FAIL — компиляция не проходит, `undefined: ValidateVideoURL`.

- [ ] **Step 3: Написать реализацию**

Создать `internal/handlers/api/url.go`:

```go
package api

import (
	"net"
	"net/url"

	"github.com/pkg/errors"
)

// ErrInvalidURL means the request URL is malformed or does not use https.
var ErrInvalidURL = errors.New("url must be a valid https:// address")

// ErrForbiddenHost means the URL points at a loopback, private or link-local address.
var ErrForbiddenHost = errors.New("url host is not allowed")

// ValidateVideoURL checks the URL before it is handed to yt-dlp. DNS is not
// resolved on purpose: a domain pointing at 127.0.0.1 passes this check. See the
// design doc for why that trade-off is acceptable here.
func ValidateVideoURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return ErrInvalidURL
	}

	host := parsed.Hostname()
	if host == "" {
		return ErrInvalidURL
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return ErrForbiddenHost
	}

	return nil
}
```

- [ ] **Step 4: Убедиться, что тест проходит**

Run: `go test ./internal/handlers/api/ -run TestValidateVideoURL -v`
Expected: PASS, все 10 подтестов зелёные.

- [ ] **Step 5: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/handlers/api/url.go internal/handlers/api/url_test.go
git commit -m "feat(api): add video URL validation"
```

---

### Task 2: JSON-ответ об ошибке и Bearer-авторизация

**Files:**
- Create: `internal/handlers/api/response.go`
- Create: `internal/handlers/api/auth.go`
- Test: `internal/handlers/api/auth_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `func WriteError(w http.ResponseWriter, code int, message string)`, `func BearerAuth(token string, next http.Handler) http.Handler` в пакете `api`.

- [ ] **Step 1: Написать падающий тест**

Создать `internal/handlers/api/auth_test.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuth(t *testing.T) {
	tests := []struct {
		name       string
		header     string
		wantStatus int
		wantNext   bool
	}{
		{"valid token", "Bearer secret", http.StatusOK, true},
		{"missing header", "", http.StatusUnauthorized, false},
		{"wrong scheme", "Basic secret", http.StatusUnauthorized, false},
		{"wrong token", "Bearer nope", http.StatusUnauthorized, false},
		{"token as prefix only", "Bearer secretsecret", http.StatusUnauthorized, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/api/download", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()

			BearerAuth("secret", next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNext {
				t.Fatalf("next handler called = %v, want %v", called, tt.wantNext)
			}
		})
	}
}

func TestBearerAuthWritesJSONError(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	req := httptest.NewRequest(http.MethodPost, "/api/download", nil)
	rec := httptest.NewRecorder()

	BearerAuth("secret", next).ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if body.Error == "" {
		t.Fatal("error field is empty")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/handlers/api/ -run TestBearerAuth -v`
Expected: FAIL — `undefined: BearerAuth`.

- [ ] **Step 3: Написать реализацию**

Создать `internal/handlers/api/response.go`:

```go
package api

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

// WriteError sends a JSON error body with the given status code.
func WriteError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}
```

Создать `internal/handlers/api/auth.go`:

```go
package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const bearerPrefix = "Bearer "

// BearerAuth rejects every request that does not carry the exact API token.
func BearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		got := strings.TrimPrefix(header, bearerPrefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/handlers/api/ -v`
Expected: PASS — тесты Task 1 и Task 2 зелёные.

- [ ] **Step 5: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/handlers/api/auth.go internal/handlers/api/response.go internal/handlers/api/auth_test.go
git commit -m "feat(api): add bearer auth middleware and json error responses"
```

---

### Task 3: Заголовок Content-Disposition для не-ASCII имён

**Files:**
- Create: `internal/handlers/api/disposition.go`
- Test: `internal/handlers/api/disposition_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `func contentDisposition(filename string) string` (не экспортируется, используется в Task 7).

- [ ] **Step 1: Написать падающий тест**

Создать `internal/handlers/api/disposition_test.go`:

```go
package api

import (
	"net/url"
	"strings"
	"testing"
)

func TestContentDispositionASCIIName(t *testing.T) {
	got := contentDisposition("TikTok - Something.mp4")
	want := `attachment; filename="TikTok - Something.mp4"; filename*=UTF-8''TikTok%20-%20Something.mp4`
	if got != want {
		t.Fatalf("contentDisposition() =\n%s\nwant\n%s", got, want)
	}
}

func TestContentDispositionNonASCIIName(t *testing.T) {
	original := "Видео 😀.mp4"
	got := contentDisposition(original)

	if !strings.HasPrefix(got, `attachment; filename="`) {
		t.Fatalf("unexpected prefix: %s", got)
	}

	fallback := got[len(`attachment; filename="`):strings.Index(got, `"; filename*=`)]
	for _, r := range fallback {
		if r > 127 {
			t.Fatalf("ascii fallback contains non-ascii rune %q: %s", r, fallback)
		}
	}
	if !strings.HasSuffix(fallback, ".mp4") {
		t.Fatalf("ascii fallback lost the extension: %s", fallback)
	}

	const marker = "filename*=UTF-8''"
	encoded := got[strings.Index(got, marker)+len(marker):]
	decoded, err := url.PathUnescape(encoded)
	if err != nil {
		t.Fatalf("filename* is not valid percent-encoding: %v", err)
	}
	if decoded != original {
		t.Fatalf("filename* decoded to %q, want %q", decoded, original)
	}
}

func TestContentDispositionQuotesAreStripped(t *testing.T) {
	got := contentDisposition(`we"ird\name.mp4`)
	fallback := got[len(`attachment; filename="`):strings.Index(got, `"; filename*=`)]
	if strings.ContainsAny(fallback, `"\`) {
		t.Fatalf("ascii fallback must not contain quotes or backslashes: %s", fallback)
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/handlers/api/ -run TestContentDisposition -v`
Expected: FAIL — `undefined: contentDisposition`.

- [ ] **Step 3: Написать реализацию**

Создать `internal/handlers/api/disposition.go`:

```go
package api

import (
	"fmt"
	"net/url"
	"strings"
)

// contentDisposition builds a header value that survives non-ASCII titles: an
// ASCII fallback for the filename parameter and an RFC 5987 form with the real
// name. Video titles routinely contain Cyrillic and emoji.
func contentDisposition(filename string) string {
	var ascii strings.Builder
	for _, r := range filename {
		if r < 128 && r != '"' && r != '\\' {
			ascii.WriteRune(r)
			continue
		}
		ascii.WriteByte('_')
	}

	fallback := ascii.String()
	if strings.Trim(fallback, "_") == "" {
		fallback = "video.mp4"
	}

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, url.PathEscape(filename))
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/handlers/api/ -v`
Expected: PASS.

- [ ] **Step 5: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/handlers/api/disposition.go internal/handlers/api/disposition_test.go
git commit -m "feat(api): build content-disposition for non-ascii filenames"
```

---

### Task 4: Разбор отказов yt-dlp по лимитам

yt-dlp при срабатывании `--max-filesize` и `--match-filter` не скачивает файл, но завершается штатно. Отличить это от настоящей ошибки можно только по выводу.

**Files:**
- Create: `internal/services/video_manager/errors.go`
- Test: `internal/services/video_manager/errors_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `var ErrTooLarge error`, `var ErrFiltered error`, `func classifyRejection(output string) error` в пакете `video_manager`.

- [ ] **Step 1: Проверить реальные строки yt-dlp**

Прогнать вручную и посмотреть, что печатает установленный yt-dlp (подстроки ниже взяты из его исходников, но версия могла поменять формулировку):

```bash
yt-dlp --no-progress --max-filesize 1k "https://www.youtube.com/watch?v=aqz-KE-bpKQ" 2>&1 | tail -3
```

```bash
yt-dlp --no-progress --match-filter "duration < 1" "https://www.youtube.com/watch?v=aqz-KE-bpKQ" 2>&1 | tail -3
```

Ожидается вывод, содержащий `larger than max-filesize` в первом случае и `does not pass filter` во втором. Если формулировки отличаются — использовать фактические подстроки в тесте и в реализации ниже.

- [ ] **Step 2: Написать падающий тест**

Создать `internal/services/video_manager/errors_test.go`:

```go
package video_manager

import (
	"testing"

	"github.com/pkg/errors"
)

func TestClassifyRejection(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   error
	}{
		{
			name:   "max filesize",
			output: "[download] File is larger than max-filesize (91234567 bytes > 524288000 bytes). Aborting download",
			want:   ErrTooLarge,
		},
		{
			name:   "match filter",
			output: `[youtube] abc: does not pass filter (duration < 900), skipping ..`,
			want:   ErrFiltered,
		},
		{
			name:   "unrelated failure",
			output: "ERROR: [youtube] abc: Video unavailable",
			want:   nil,
		},
		{
			name:   "empty output",
			output: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyRejection(tt.output)
			if !errors.Is(got, tt.want) {
				t.Fatalf("classifyRejection() = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: Убедиться, что тест падает**

Run: `go test ./internal/services/video_manager/ -run TestClassifyRejection -v`
Expected: FAIL — `undefined: classifyRejection`.

- [ ] **Step 4: Написать реализацию**

Создать `internal/services/video_manager/errors.go`:

```go
package video_manager

import (
	"strings"

	"github.com/pkg/errors"
)

// ErrTooLarge means yt-dlp refused the video because of --max-filesize.
var ErrTooLarge = errors.New("video is larger than the allowed size")

// ErrFiltered means yt-dlp refused the video because of --match-filter.
var ErrFiltered = errors.New("video does not pass the duration filter")

// classifyRejection maps yt-dlp output to a typed error when the run produced no
// file. yt-dlp exits successfully in both cases, so its output is the only signal.
func classifyRejection(output string) error {
	switch {
	case strings.Contains(output, "larger than max-filesize"):
		return ErrTooLarge
	case strings.Contains(output, "does not pass filter"):
		return ErrFiltered
	default:
		return nil
	}
}
```

- [ ] **Step 5: Убедиться, что тест проходит**

Run: `go test ./internal/services/video_manager/ -run TestClassifyRejection -v`
Expected: PASS, 4 подтеста зелёные.

- [ ] **Step 6: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/services/video_manager/errors.go internal/services/video_manager/errors_test.go
git commit -m "feat(video): classify yt-dlp size and duration rejections"
```

---

### Task 5: Проверка кодеков через ffprobe

**Files:**
- Create: `internal/services/video_manager/probe.go`
- Test: `internal/services/video_manager/probe_test.go`

**Interfaces:**
- Consumes: ничего.
- Produces: `func probeNeedsTranscode(ctx context.Context, path string) (bool, error)` в пакете `video_manager`. В Task 6 она станет телом метода интерфейса.

- [ ] **Step 1: Написать падающий тест**

Тест интеграционный: генерирует файлы настоящим ffmpeg и пропускается, если ffmpeg не установлен. Кодек `mpeg4` выбран для отрицательного случая потому, что он встроен в ffmpeg и есть в любой сборке.

Создать `internal/services/video_manager/probe_test.go`:

```go
package video_manager

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
)

func runFFmpeg(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("ffmpeg", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg %v failed: %v\n%s", args, err, out)
	}
}

func TestProbeNeedsTranscode(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	dir := t.TempDir()
	compatible := filepath.Join(dir, "compatible.mp4")
	wrongCodec := filepath.Join(dir, "wrong-codec.mp4")
	noAudio := filepath.Join(dir, "no-audio.mp4")

	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "libx264", "-c:a", "aac", "-pix_fmt", "yuv420p", "-y", compatible)

	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1",
		"-c:v", "mpeg4", "-c:a", "aac", "-y", wrongCodec)

	runFFmpeg(t,
		"-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-y", noAudio)

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"h264 and aac in mp4 needs nothing", compatible, false},
		{"mpeg4 video needs transcode", wrongCodec, true},
		{"missing audio track needs transcode", noAudio, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := probeNeedsTranscode(context.Background(), tt.path)
			if err != nil {
				t.Fatalf("probeNeedsTranscode() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("probeNeedsTranscode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbeNeedsTranscodeMissingFile(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe is not installed")
	}

	if _, err := probeNeedsTranscode(context.Background(), filepath.Join(t.TempDir(), "nope.mp4")); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}
```

- [ ] **Step 2: Убедиться, что тест падает**

Run: `go test ./internal/services/video_manager/ -run TestProbeNeedsTranscode -v`
Expected: FAIL — `undefined: probeNeedsTranscode`.

- [ ] **Step 3: Написать реализацию**

Создать `internal/services/video_manager/probe.go`:

```go
package video_manager

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

type probeOutput struct {
	Format struct {
		FormatName string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecName string `json:"codec_name"`
		CodecType string `json:"codec_type"`
	} `json:"streams"`
}

// probeNeedsTranscode reports whether the file has to go through ffmpeg before
// Apple Photos will accept it: mp4 container, H.264 video, AAC audio. yt-dlp
// already returns such a file for most short-form sources, and skipping the
// transcode saves the caller tens of seconds.
func probeNeedsTranscode(ctx context.Context, path string) (bool, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=format_name:stream=codec_name,codec_type",
		"-of", "json",
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return false, errors.Wrap(err, "failed to run ffprobe")
	}

	var probe probeOutput
	if err := json.Unmarshal(out, &probe); err != nil {
		return false, errors.Wrap(err, "failed to parse ffprobe output")
	}

	if !strings.Contains(probe.Format.FormatName, "mp4") {
		return true, nil
	}

	var hasVideo, hasAudio bool
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			if stream.CodecName != "h264" {
				return true, nil
			}
			hasVideo = true
		case "audio":
			if stream.CodecName != "aac" {
				return true, nil
			}
			hasAudio = true
		}
	}

	return !hasVideo || !hasAudio, nil
}
```

- [ ] **Step 4: Убедиться, что тесты проходят**

Run: `go test ./internal/services/video_manager/ -v`
Expected: PASS — тесты Task 4 и Task 5 зелёные.

- [ ] **Step 5: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/services/video_manager/probe.go internal/services/video_manager/probe_test.go
git commit -m "feat(video): detect whether a file needs transcoding via ffprobe"
```

---

### Task 6: Методы video_manager с контекстом и каталогом

Скачивание переезжает на per-call `*ytdlp.Command` и явный каталог: сейчас все загрузки делят одно поле структуры и пишут в рабочую директорию по общему шаблону имени, из-за чего два одинаковых URL одновременно спотыкаются о `NoOverwrites`.

Поведение бота при этом не меняется: он как транскодировал каждое видео, так и продолжает. Пропуск транскода добавляется только в API-пути (Task 7).

**Files:**
- Modify: `internal/services/video_manager/service.go` (полностью переписывается)
- Modify: `internal/handlers/message/http/hander.go` (переход на новые методы)

**Interfaces:**
- Consumes: `classifyRejection`, `ErrTooLarge`, `ErrFiltered` (Task 4); `probeNeedsTranscode` (Task 5).
- Produces: интерфейс `video_manager.VideoManager` с методами
  `DownloadVideoTo(ctx context.Context, url, dir string) (string, error)`,
  `NeedsTranscode(ctx context.Context, path string) (bool, error)`,
  `TranscodeVideoTo(ctx context.Context, inputPath, dir string) (string, error)`.
  Методы `DownloadVideo`, `TranscodeVideo` и `DeleteVideo` удаляются: очистка теперь делается через `os.RemoveAll` каталога запроса.

- [ ] **Step 1: Переписать service.go**

Заменить содержимое `internal/services/video_manager/service.go`:

```go
package video_manager

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/pkg/errors"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/infrastructure/metrics"
)

const (
	maxFileSize    = "500M"
	durationFilter = "duration < 900"
	outputTemplate = "%(extractor)s - %(title).100B.%(ext)s"
	installTimeout = 5 * time.Minute
)

type VideoManager interface {
	DownloadVideoTo(ctx context.Context, url, dir string) (string, error)
	NeedsTranscode(ctx context.Context, path string) (bool, error)
	TranscodeVideoTo(ctx context.Context, inputPath, dir string) (string, error)
}

type DefaultVideoManager struct {
	log interfaces.Logger
}

func New(log interfaces.Logger) (VideoManager, error) {
	log.Info("Checking ytdlp lib installed")
	installCtx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()
	if _, err := ytdlp.Install(installCtx, nil); err != nil {
		return nil, errors.Wrap(err, "failed to install ytdlp")
	}
	log.Info("ytdlp lib installed")

	log.Info("Checking ffmpeg installed")
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, errors.Wrap(err, "ffmpeg not found in PATH")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return nil, errors.Wrap(err, "ffprobe not found in PATH")
	}
	log.Info("ffmpeg and ffprobe found")

	return DefaultVideoManager{log: log}, nil
}

// DownloadVideoTo downloads the video into dir and returns the resulting path.
// The command is built per call: a shared builder would race between concurrent
// downloads.
func (d DefaultVideoManager) DownloadVideoTo(ctx context.Context, url, dir string) (string, error) {
	d.log.Info("Downloading video from: " + url)

	dl := ytdlp.New().
		PrintJSON().
		NoProgress().
		FormatSort("res,ext:mp4:m4a").
		NoPlaylist().
		NoOverwrites().
		MaxFileSize(maxFileSize).
		MatchFilters(durationFilter).
		Paths(dir).
		Output(outputTemplate)

	start := time.Now()
	result, err := dl.Run(ctx, url)
	metrics.DownloadDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DownloadTotal.WithLabelValues("error").Inc()
		d.log.WithError(err).Warn("Failed to download video from: " + url)
		if result != nil {
			if rejected := classifyRejection(result.Stdout + "\n" + result.Stderr); rejected != nil {
				return "", rejected
			}
		}
		return "", errors.Wrap(err, "failed to run yt-dlp")
	}

	infos, err := result.GetExtractedInfo()
	if err != nil {
		metrics.DownloadTotal.WithLabelValues("error").Inc()
		return "", errors.Wrap(err, "failed to parse yt-dlp output")
	}

	for _, info := range infos {
		if info.Filename != nil {
			metrics.DownloadTotal.WithLabelValues("success").Inc()
			d.log.Info("Successfully downloaded video from: " + url + " to: " + *info.Filename)
			return *info.Filename, nil
		}
	}

	metrics.DownloadTotal.WithLabelValues("error").Inc()
	if rejected := classifyRejection(result.Stdout + "\n" + result.Stderr); rejected != nil {
		return "", rejected
	}
	return "", errors.New("failed to get video filename")
}

// NeedsTranscode reports whether the file must be re-encoded before it can be
// handed to a client that expects H.264/AAC in mp4.
func (d DefaultVideoManager) NeedsTranscode(ctx context.Context, path string) (bool, error) {
	return probeNeedsTranscode(ctx, path)
}

// TranscodeVideoTo re-encodes the file into dir and returns the new path.
func (d DefaultVideoManager) TranscodeVideoTo(ctx context.Context, inputPath, dir string) (string, error) {
	f, err := os.CreateTemp(dir, "tgvd-*.tc.mp4")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file for transcoding")
	}
	if err = f.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close temp file")
	}
	outputPath := f.Name()

	d.log.Info("Transcoding video: " + inputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-y",
		outputPath,
	)

	start := time.Now()
	out, err := cmd.CombinedOutput()
	metrics.TranscodeDuration.Observe(time.Since(start).Seconds())
	if err != nil {
		metrics.TranscodeTotal.WithLabelValues("error").Inc()
		d.log.WithError(err).WithField("ffmpeg_output", string(out)).Warn("ffmpeg failed")
		return "", errors.New("ffmpeg transcoding failed")
	}

	metrics.TranscodeTotal.WithLabelValues("success").Inc()
	d.log.Info("Transcoded video to: " + outputPath)
	return outputPath, nil
}
```

- [ ] **Step 2: Перевести бота на новые методы**

Заменить в `internal/handlers/message/http/hander.go` импорты и тело `HandleMessage` между отправкой «Downloading video...» и отправкой файла. Вместо двух `defer` с `DeleteVideo` — один временный каталог на сообщение:

```go
const botDownloadTimeout = 10 * time.Minute
```

```go
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

	ctx, cancel := context.WithTimeout(context.Background(), botDownloadTimeout)
	defer cancel()

	videoPath, err := h.videoDownloader.DownloadVideoTo(ctx, message.Text, dir)
	if err != nil {
		h.log.WithError(err).Warn("failed to download video")
		_, err = h.messageSender.ReplyTo(message, "failed to download video: "+err.Error(), false)
		return err
	}

	if err = h.messageSender.EditMessage(message.Chat.ID, msg.MessageID, "Transcoding video..."); err != nil {
		return err
	}

	transcodedPath, err := h.videoDownloader.TranscodeVideoTo(ctx, videoPath, dir)
	if err != nil {
		h.log.WithError(err).Warn("failed to transcode video")
		_, _ = h.messageSender.ReplyTo(message, "failed to transcode video: "+err.Error(), false)
		return err
	}
```

Дальше идёт существующий блок с `VideoReplyTo(message, transcodedPath)` и метриками аплоада — он не меняется. Импорты `context`, `os` и `time` добавить, если их нет.

- [ ] **Step 3: Проверить сборку и тесты**

Run: `go build ./... && go test ./...`
Expected: сборка проходит, тесты Task 1–5 зелёные.

- [ ] **Step 4: Проверить, что yt-dlp возвращает путь внутри каталога**

Написать и выполнить одноразовую проверку (файл потом удалить):

```bash
cat > /tmp/paths_check.go <<'EOF'
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lrstanley/go-ytdlp"
)

func main() {
	dir, _ := os.MkdirTemp("", "paths-check-*")
	defer os.RemoveAll(dir)

	// Подставить ссылку на любой короткий ролик (10–60 секунд): Shorts, Reels, TikTok.
	res, err := ytdlp.New().PrintJSON().NoProgress().NoPlaylist().
		Paths(dir).Output("%(extractor)s - %(title).100B.%(ext)s").
		Run(context.Background(), "https://www.youtube.com/shorts/tPEE9ZwTmy0")
	if err != nil {
		fmt.Println("run error:", err)
		return
	}
	infos, _ := res.GetExtractedInfo()
	for _, i := range infos {
		if i.Filename != nil {
			fmt.Println("filename:", *i.Filename)
		}
	}
}
EOF
go run /tmp/paths_check.go
```

Expected: напечатанный путь начинается с временного каталога. Если yt-dlp вернул голое имя файла без каталога — в `DownloadVideoTo` вернуть `filepath.Join(dir, *info.Filename)` для относительных путей и зафиксировать это в коде комментарием.

- [ ] **Step 5: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/services/video_manager/service.go internal/handlers/message/http/hander.go
git commit -m "refactor(video): download into a per-request dir with an explicit context"
```

---

### Task 7: HTTP-хэндлер /api/download

**Files:**
- Create: `internal/handlers/api/download.go`
- Test: `internal/handlers/api/download_test.go`
- Modify: `internal/infrastructure/metrics/metrics.go` (добавить счётчик)

**Interfaces:**
- Consumes: `ValidateVideoURL`, `ErrInvalidURL`, `ErrForbiddenHost` (Task 1); `WriteError` (Task 2); `contentDisposition` (Task 3); `video_manager.VideoManager`, `ErrTooLarge`, `ErrFiltered` (Task 4, 6).
- Produces: `func NewDownloadHandler(log interfaces.Logger, manager video_manager.VideoManager, maxConcurrent int) *DownloadHandler` и метод `ServeHTTP`; `metrics.APIRequests`.

- [ ] **Step 1: Добавить метрику**

В `internal/infrastructure/metrics/metrics.go` добавить в блок `var (...)`:

```go
	// APIRequests counts HTTP download API responses by status.
	APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "api_requests_total",
		Help:      "Total number of download API responses by status",
	}, []string{"status"})
```

- [ ] **Step 2: Написать падающий тест**

Создать `internal/handlers/api/download_test.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"tg-video-downloader/internal/infrastructure/logger"
	"tg-video-downloader/internal/services/video_manager"
)

type fakeVideoManager struct {
	downloadErr   error
	downloadBody  string
	needs         bool
	needsErr      error
	transcodeErr  error
	transcodeBody string

	gotDir       string
	transcodeHit bool
}

func (f *fakeVideoManager) DownloadVideoTo(ctx context.Context, url, dir string) (string, error) {
	f.gotDir = dir
	if f.downloadErr != nil {
		return "", f.downloadErr
	}
	path := filepath.Join(dir, "Extractor - Title.mp4")
	if err := os.WriteFile(path, []byte(f.downloadBody), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (f *fakeVideoManager) NeedsTranscode(ctx context.Context, path string) (bool, error) {
	if f.needsErr != nil {
		return false, f.needsErr
	}
	return f.needs, nil
}

func (f *fakeVideoManager) TranscodeVideoTo(ctx context.Context, inputPath, dir string) (string, error) {
	f.transcodeHit = true
	if f.transcodeErr != nil {
		return "", f.transcodeErr
	}
	path := filepath.Join(dir, "transcoded.mp4")
	if err := os.WriteFile(path, []byte(f.transcodeBody), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func newTestHandler(manager video_manager.VideoManager) *DownloadHandler {
	return NewDownloadHandler(logger.GetLogger(), manager, 2)
}

func post(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/download", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestDownloadHandlerServesFile(t *testing.T) {
	manager := &fakeVideoManager{downloadBody: "fake-mp4-bytes"}
	rec := post(t, newTestHandler(manager), `{"url":"https://www.tiktok.com/@user/video/123"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "video/mp4" {
		t.Fatalf("Content-Type = %q, want video/mp4", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "Extractor - Title.mp4") {
		t.Fatalf("Content-Disposition = %q", got)
	}
	if rec.Body.String() != "fake-mp4-bytes" {
		t.Fatalf("body = %q, want the downloaded file contents", rec.Body.String())
	}
	if manager.transcodeHit {
		t.Fatal("transcode must be skipped when NeedsTranscode reports false")
	}
	if manager.gotDir == "" {
		t.Fatal("handler did not pass a temp dir to the manager")
	}
	if _, err := os.Stat(manager.gotDir); !os.IsNotExist(err) {
		t.Fatalf("temp dir %s was not cleaned up", manager.gotDir)
	}
}

func TestDownloadHandlerTranscodesWhenNeeded(t *testing.T) {
	manager := &fakeVideoManager{downloadBody: "raw", needs: true, transcodeBody: "converted"}
	rec := post(t, newTestHandler(manager), `{"url":"https://www.tiktok.com/@user/video/123"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !manager.transcodeHit {
		t.Fatal("transcode was not called")
	}
	if rec.Body.String() != "converted" {
		t.Fatalf("body = %q, want the transcoded file contents", rec.Body.String())
	}
}

func TestDownloadHandlerErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		manager    *fakeVideoManager
		body       string
		method     string
		wantStatus int
	}{
		{
			name:       "video too large",
			manager:    &fakeVideoManager{downloadErr: video_manager.ErrTooLarge},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "video filtered by duration",
			manager:    &fakeVideoManager{downloadErr: video_manager.ErrFiltered},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "download deadline",
			manager:    &fakeVideoManager{downloadErr: context.DeadlineExceeded},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusGatewayTimeout,
		},
		{
			name:       "yt-dlp failure",
			manager:    &fakeVideoManager{downloadErr: errors.New("boom")},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "probe failure",
			manager:    &fakeVideoManager{needsErr: errors.New("boom")},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "transcode failure",
			manager:    &fakeVideoManager{needs: true, transcodeErr: errors.New("boom")},
			body:       `{"url":"https://example.com/v"}`,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "malformed json",
			manager:    &fakeVideoManager{},
			body:       `{"url":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "non https url",
			manager:    &fakeVideoManager{},
			body:       `{"url":"http://example.com/v"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "loopback url",
			manager:    &fakeVideoManager{},
			body:       `{"url":"https://127.0.0.1/v"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(t, newTestHandler(tt.manager), tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var body struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode error body: %v", err)
			}
			if body.Error == "" {
				t.Fatal("error field is empty")
			}
		})
	}
}

func TestDownloadHandlerRejectsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/download", nil)
	rec := httptest.NewRecorder()
	newTestHandler(&fakeVideoManager{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}
```

- [ ] **Step 3: Убедиться, что тесты падают**

Run: `go test ./internal/handlers/api/ -run TestDownloadHandler -v`
Expected: FAIL — `undefined: NewDownloadHandler`, `undefined: DownloadHandler`.

- [ ] **Step 4: Написать реализацию**

Создать `internal/handlers/api/download.go`:

```go
package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
	"tg-video-downloader/internal/infrastructure/metrics"
	"tg-video-downloader/internal/services/video_manager"
)

const (
	requestTimeout = 5 * time.Minute
	maxBodySize    = 8 << 10
)

type downloadRequest struct {
	URL string `json:"url"`
}

// DownloadHandler downloads a video and streams it back in the response body.
type DownloadHandler struct {
	manager video_manager.VideoManager
	log     interfaces.Logger
	sem     chan struct{}
}

func NewDownloadHandler(log interfaces.Logger, manager video_manager.VideoManager, maxConcurrent int) *DownloadHandler {
	return &DownloadHandler{
		manager: manager,
		log:     log,
		sem:     make(chan struct{}, maxConcurrent),
	}
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.fail(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	select {
	case h.sem <- struct{}{}:
		defer func() { <-h.sem }()
	default:
		h.fail(w, http.StatusTooManyRequests, "another download is in progress")
		return
	}

	var req downloadRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, maxBodySize)).Decode(&req); err != nil {
		h.fail(w, http.StatusBadRequest, "invalid json body")
		return
	}

	if err := ValidateVideoURL(req.URL); err != nil {
		h.fail(w, http.StatusBadRequest, err.Error())
		return
	}

	dir, err := os.MkdirTemp("", "tgvd-api-*")
	if err != nil {
		h.log.WithError(err).Error("failed to create temp dir")
		h.fail(w, http.StatusInternalServerError, "failed to create temp dir")
		return
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			h.log.WithError(err).Warn("failed to clean up temp dir")
		}
	}()

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	path, err := h.manager.DownloadVideoTo(ctx, req.URL, dir)
	if err != nil {
		h.failDownload(w, err)
		return
	}

	needs, err := h.manager.NeedsTranscode(ctx, path)
	if err != nil {
		h.log.WithError(err).Warn("failed to probe video")
		h.fail(w, http.StatusBadGateway, "failed to probe video")
		return
	}

	if needs {
		path, err = h.manager.TranscodeVideoTo(ctx, path, dir)
		if err != nil {
			h.log.WithError(err).Warn("failed to transcode video")
			h.fail(w, http.StatusBadGateway, "failed to transcode video")
			return
		}
	}

	h.serveFile(w, path)
}

func (h *DownloadHandler) serveFile(w http.ResponseWriter, path string) {
	file, err := os.Open(path)
	if err != nil {
		h.log.WithError(err).Error("failed to open downloaded video")
		h.fail(w, http.StatusInternalServerError, "failed to open video")
		return
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		h.log.WithError(err).Error("failed to stat downloaded video")
		h.fail(w, http.StatusInternalServerError, "failed to stat video")
		return
	}

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	w.Header().Set("Content-Disposition", contentDisposition(filepath.Base(path)))
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		h.log.WithError(err).Warn("failed to stream video to the client")
		metrics.APIRequests.WithLabelValues("stream_error").Inc()
		return
	}

	metrics.APIRequests.WithLabelValues("200").Inc()
}

func (h *DownloadHandler) failDownload(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, video_manager.ErrTooLarge):
		h.fail(w, http.StatusRequestEntityTooLarge, "video is larger than the limit")
	case errors.Is(err, video_manager.ErrFiltered):
		h.fail(w, http.StatusRequestEntityTooLarge, "video is longer than the limit")
	case errors.Is(err, context.DeadlineExceeded):
		h.fail(w, http.StatusGatewayTimeout, "download timed out")
	default:
		h.log.WithError(err).Warn("failed to download video")
		h.fail(w, http.StatusBadGateway, "failed to download video")
	}
}

func (h *DownloadHandler) fail(w http.ResponseWriter, code int, message string) {
	metrics.APIRequests.WithLabelValues(strconv.Itoa(code)).Inc()
	WriteError(w, code, message)
}
```

- [ ] **Step 5: Убедиться, что тесты проходят**

Run: `go test ./internal/handlers/api/ -v`
Expected: PASS — все тесты пакета зелёные.

- [ ] **Step 6: Коммит**

```bash
gofmt -l . && golangci-lint run
git add internal/handlers/api/download.go internal/handlers/api/download_test.go internal/infrastructure/metrics/metrics.go
git commit -m "feat(api): add POST /api/download handler"
```

---

### Task 8: Проводка в main.go

**Files:**
- Modify: `cmd/service/main.go`
- Modify: `.env.example`

**Interfaces:**
- Consumes: `api.NewDownloadHandler`, `api.BearerAuth` (Task 2, 7); `video_manager.New` (Task 6).
- Produces: работающий эндпоинт `POST /api/download` на порту 9900.

- [ ] **Step 1: Зарегистрировать роут**

В `cmd/service/main.go` добавить импорт `"tg-video-downloader/internal/handlers/api"` и константу рядом с остальными:

```go
const apiMaxConcurrent = 2
```

После чтения `METRICS_*` прочитать токен:

```go
	apiToken := os.Getenv("API_TOKEN")
```

Блок с `mux` заменить на:

```go
	mux := netHttp.NewServeMux()
	mux.Handle("/metrics", metricsBasicAuth(metricsUsername, metricsPassword, promhttp.Handler()))

	if apiToken == "" {
		log.Warn("API_TOKEN is not set, POST /api/download is disabled")
	} else {
		downloadHandler := api.NewDownloadHandler(log, videoDownloader, apiMaxConcurrent)
		mux.Handle("/api/download", api.BearerAuth(apiToken, downloadHandler))
		log.Info("Download API is enabled on POST /api/download")
	}

	srv := &netHttp.Server{
		Addr:              ":9900",
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
```

`WriteTimeout` не задавать: он обрывает отдачу большого файла. Ограничение по времени живёт в контексте запроса внутри хэндлера.

- [ ] **Step 2: Добавить переменную в .env.example**

Дописать строку в `.env.example`:

```
API_TOKEN=your_api_token_here
```

- [ ] **Step 3: Проверить сборку и тесты**

Run: `go build ./... && go test ./... && gofmt -l . && golangci-lint run`
Expected: всё зелёное, `gofmt -l .` ничего не печатает.

- [ ] **Step 4: Проверить ручку живьём**

Заполнить `.env` (включая `API_TOKEN=localtest`), запустить сервис:

```bash
go run ./cmd/service/main.go
```

В другом терминале — успешный сценарий. Вместо ссылки ниже подставить любой короткий
ролик (10–60 секунд), до которого дотягивается сеть машины:

```bash
curl -sS -D - -X POST http://localhost:9900/api/download \
  -H "Authorization: Bearer localtest" \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.youtube.com/shorts/tPEE9ZwTmy0"}' \
  -o /tmp/out.mp4
```

Expected: `200`, заголовки `Content-Type: video/mp4` и `Content-Disposition`, файл открывается плеером:

```bash
ffprobe -v error -show_entries format=format_name -show_entries stream=codec_name /tmp/out.mp4
```

Проверить отказы:

```bash
curl -sS -o - -w '\n%{http_code}\n' -X POST http://localhost:9900/api/download \
  -H "Content-Type: application/json" -d '{"url":"https://example.com/v"}'
```

Expected: `401` и JSON с полем `error`.

```bash
curl -sS -o - -w '\n%{http_code}\n' -X POST http://localhost:9900/api/download \
  -H "Authorization: Bearer localtest" \
  -H "Content-Type: application/json" -d '{"url":"http://example.com/v"}'
```

Expected: `400`.

Убедиться, что временные каталоги не копятся:

```bash
ls -d ${TMPDIR:-/tmp}/tgvd-api-* 2>/dev/null || echo "no leftovers"
```

Expected: `no leftovers`.

- [ ] **Step 5: Убедиться, что ffprobe есть в образе**

`video_manager.New` теперь падает при старте, если `ffprobe` не найден, поэтому его
наличие в образе нужно подтвердить до деплоя. Пакет `ffmpeg` в alpine обычно тянет
`ffprobe` за собой, но проверить это дешевле, чем ловить упавший контейнер:

```bash
docker run --rm alpine:latest sh -c "apk add --no-cache ffmpeg >/dev/null && which ffprobe"
```

Expected: `/usr/bin/ffprobe`. Если команда ничего не вывела — добавить `ffprobe` в
`apk --no-cache add` в `Dockerfile` и включить файл в коммит ниже.

Затем собрать образ целиком и убедиться, что сервис стартует:

```bash
docker compose up --build
```

Expected: в логах есть `ffmpeg and ffprobe found` и `Download API is enabled on POST /api/download`.

- [ ] **Step 6: Коммит**

```bash
git add cmd/service/main.go .env.example
git commit -m "feat: wire POST /api/download into the service"
```

---

### Task 9: Документация и инструкция для Shortcuts

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`

**Interfaces:**
- Consumes: готовый эндпоинт из Task 8.
- Produces: ничего для кода.

- [ ] **Step 1: Дополнить README**

В таблицу переменных окружения добавить строку:

```markdown
| `API_TOKEN` | Bearer token for `POST /api/download`; the endpoint is disabled when unset |
```

Добавить раздел после описания метрик:

````markdown
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
| 413 | video is over 500 MB or longer than 15 minutes |
| 429 | two downloads are already running |
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
````

- [ ] **Step 2: Обновить CLAUDE.md**

В таблицу переменных добавить `API_TOKEN`. В блок архитектуры добавить строки:

```
internal/handlers/api/
  download.go                — POST /api/download: validate → download → probe → transcode → stream file
  auth.go                    — bearer token middleware
  url.go                     — URL validation (https only, no private hosts)
  disposition.go             — RFC 5987 Content-Disposition
```

В разделе про метрики дописать: `POST /api/download` живёт на том же порту, что и `/metrics`, и включается только при заданном `API_TOKEN`.

Заменить фразу «No tests exist in the codebase» на описание фактического состояния: тесты есть в `internal/handlers/api` и `internal/services/video_manager`, запускаются через `go test ./...`, часть из них пропускается без ffmpeg.

- [ ] **Step 3: Проверить документацию на соответствие коду**

Run: `go test ./... && go build ./...`
Сверить глазами: значения лимитов в README (500 МБ, 15 минут, 2 загрузки, 5 минут) совпадают с константами в `internal/services/video_manager/service.go` и `internal/handlers/api/download.go`.

- [ ] **Step 4: Коммит**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document the download API and the Shortcuts setup"
```

---

## Проверка перед PR

- [ ] `go build ./...` — успешно
- [ ] `go test ./...` — успешно
- [ ] `gofmt -l .` — пусто
- [ ] `golangci-lint run` — без замечаний
- [ ] Бот в Telegram по-прежнему принимает ссылку и присылает видео (ручной прогон с настоящим токеном)
- [ ] `curl` на `/api/download` возвращает воспроизводимый mp4
- [ ] `/metrics` отвечает и содержит `tgvd_api_requests_total`
