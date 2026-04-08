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

const downloadTimeout = 10 * time.Minute

type VideoManager interface {
	DownloadVideo(url string) (string, error)
	TranscodeVideo(inputPath string) (string, error)
	DeleteVideo(fileName string) error
}

type DefaultVideoManager struct {
	dl  *ytdlp.Command
	log interfaces.Logger
}

func New(log interfaces.Logger) (VideoManager, error) {
	log.Info("Checking ytdlp lib installed")
	installCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if _, err := ytdlp.Install(installCtx, nil); err != nil {
		return nil, errors.Wrap(err, "failed to install ytdlp")
	}
	log.Info("ytdlp lib installed")

	dl := ytdlp.New().
		PrintJSON().
		NoProgress().
		FormatSort("res,ext:mp4:m4a").
		NoPlaylist().
		NoOverwrites().
		Output("%(extractor)s - %(title)s.%(ext)s")

	return DefaultVideoManager{
		dl:  dl,
		log: log,
	}, nil
}

func (d DefaultVideoManager) DownloadVideo(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	d.log.Info("Downloading video from: " + url)

	start := time.Now()
	result, err := d.dl.Run(ctx, url)
	metrics.DownloadDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		metrics.DownloadTotal.WithLabelValues("error").Inc()
		d.log.WithError(err).Warn("Failed to download video from: " + url)
		return "", errors.Wrap(err, "failed to run yt-dlp")
	}

	infos, err := result.GetExtractedInfo()
	if err != nil {
		metrics.DownloadTotal.WithLabelValues("error").Inc()
		return "", errors.Wrap(err, "failed to parse yt-dlp output")
	}

	for _, info := range infos {
		filename := info.Filename
		if filename != nil {
			metrics.DownloadTotal.WithLabelValues("success").Inc()
			d.log.Info("Successfully downloaded video from: " + url + " to: " + *filename)
			return *filename, nil
		}
	}

	metrics.DownloadTotal.WithLabelValues("error").Inc()
	return "", errors.New("failed to get video filename")
}

func (d DefaultVideoManager) TranscodeVideo(inputPath string) (string, error) {
	f, err := os.CreateTemp("", "tgvd-*.tc.mp4")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temp file for transcoding")
	}
	f.Close()
	outputPath := f.Name()

	d.log.Info("Transcoding video: " + inputPath)

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

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

func (d DefaultVideoManager) DeleteVideo(fileName string) error {
	err := os.Remove(fileName)
	if err != nil {
		return errors.Wrap(err, "failed to delete video file")
	}
	return nil
}
