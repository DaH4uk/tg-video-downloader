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
