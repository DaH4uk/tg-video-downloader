package http

import (
	"errors"
	"testing"

	"tg-video-downloader/internal/services/video_manager"
)

func TestDownloadErrorReply(t *testing.T) {
	if got, want := downloadErrorReply(video_manager.ErrTikTokIPBlocked), "TikTok temporarily blocks this bot from accessing this video. Please try again later."; got != want {
		t.Fatalf("reply = %q, want %q", got, want)
	}
}

func TestDownloadErrorReplyIncludesOtherErrors(t *testing.T) {
	if got, want := downloadErrorReply(errors.New("boom")), "failed to download video: boom"; got != want {
		t.Fatalf("reply = %q, want %q", got, want)
	}
}
