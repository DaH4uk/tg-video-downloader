package api

import (
	"context"
	"encoding/json"
	"fmt"
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
		// cap(h.sem) is the single source of truth for the concurrency limit, so
		// the message can never drift from the actual configured value.
		h.fail(w, http.StatusTooManyRequests, fmt.Sprintf("%d downloads are already running", cap(h.sem)))
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
		h.failDownload(w, ctx, err)
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

// failDownload maps a download failure to an HTTP response. The returned error
// from exec.CommandContext does not carry context.DeadlineExceeded through its
// error chain (os/exec reports "signal: killed", and go-ytdlp's result formatting
// uses %s instead of %w), so timeout/cancellation must be read from ctx.Err()
// rather than from err itself.
func (h *DownloadHandler) failDownload(w http.ResponseWriter, ctx context.Context, err error) {
	switch {
	case errors.Is(err, video_manager.ErrTooLarge):
		h.fail(w, http.StatusRequestEntityTooLarge, "video is larger than the limit")
	case errors.Is(err, video_manager.ErrFiltered):
		h.fail(w, http.StatusRequestEntityTooLarge, "video is longer than the limit")
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		h.fail(w, http.StatusGatewayTimeout, "download timed out")
	case errors.Is(ctx.Err(), context.Canceled):
		// The client (e.g. a disconnected Shortcut) hung up before the download
		// finished. Nothing is listening for a response anymore, so don't write
		// one or count it as a gateway error; just note it happened.
		h.log.Info("client disconnected before download finished")
	default:
		h.log.WithError(err).Warn("failed to download video")
		h.fail(w, http.StatusBadGateway, "failed to download video")
	}
}

func (h *DownloadHandler) fail(w http.ResponseWriter, code int, message string) {
	metrics.APIRequests.WithLabelValues(strconv.Itoa(code)).Inc()
	WriteError(w, code, message)
}
