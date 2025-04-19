package video_manager

import (
	"context"
	"os"

	"github.com/lrstanley/go-ytdlp"
	"github.com/pkg/errors"

	"telegram-vpn-bot/internal/infrastructure/logger/interfaces"
)

type VideoManager interface {
	DownloadVideo(url string) (string, error)
	DeleteVideo(fileName string) error
}

type DefaultVideoManager struct {
	dl  *ytdlp.Command
	log interfaces.Logger
}

func New(log interfaces.Logger) VideoManager {
	log.Info("Checking ytdlp lib installed")
	// If yt-dlp isn't installed yet, download and cache it for further use.
	ytdlp.MustInstall(context.TODO(), nil)
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
	}
}

func (d DefaultVideoManager) DownloadVideo(url string) (string, error) {
	d.log.Info("Downloading video from: " + url)
	result, err := d.dl.Run(context.Background(), url)

	if err != nil {
		d.log.WithError(err).Warn("Failed to download video from: " + url)
		return "", err
	}
	infos, err := result.GetExtractedInfo()
	if err != nil {
		return "", err
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
