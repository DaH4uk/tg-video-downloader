package video_manager

import (
	"context"
	"os"
	"time"

	"github.com/lrstanley/go-ytdlp"
	"github.com/pkg/errors"

	"tg-video-downloader/internal/infrastructure/logger/interfaces"
)

const downloadTimeout = 10 * time.Minute

type VideoManager interface {
	DownloadVideo(url string) (string, error)
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
	result, err := d.dl.Run(ctx, url)
	if err != nil {
		d.log.WithError(err).Warn("Failed to download video from: " + url)
		return "", errors.Wrap(err, "failed to run yt-dlp")
	}

	infos, err := result.GetExtractedInfo()
	if err != nil {
		return "", errors.Wrap(err, "failed to parse yt-dlp output")
	}

	for _, info := range infos {
		filename := info.Filename
		if filename != nil {
			d.log.Info("Successfully downloaded video from: " + url + " to: " + *filename)
			return *filename, nil
		}
	}

	return "", errors.New("failed to get video filename")
}

func (d DefaultVideoManager) DeleteVideo(fileName string) error {
	err := os.Remove(fileName)
	if err != nil {
		return errors.Wrap(err, "failed to delete video file")
	}
	return nil
}