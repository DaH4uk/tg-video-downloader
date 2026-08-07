package video_manager

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
			// Verified empirically (live yt-dlp run) that Filename comes back as an
			// absolute path inside dir; this join is a defensive fallback in case a
			// future extractor or go-ytdlp version ever returns a relative one.
			name := *info.Filename
			if !filepath.IsAbs(name) {
				name = filepath.Join(dir, name)
			}
			d.log.Info("Successfully downloaded video from: " + url + " to: " + name)
			return name, nil
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
