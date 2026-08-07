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
