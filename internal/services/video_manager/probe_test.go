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
