package video_manager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDownloadedVideo(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "video.MP4")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, ok := findDownloadedVideo(dir)
	if !ok {
		t.Fatal("expected downloaded video to be found")
	}
	if got != videoPath {
		t.Fatalf("path = %q, want %q", got, videoPath)
	}
}

func TestFindDownloadedVideoIgnoresNonVideoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	if path, ok := findDownloadedVideo(dir); ok {
		t.Fatalf("found %q, want no video", path)
	}
}
