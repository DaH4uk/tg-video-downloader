package video_manager

import (
	"strings"

	"github.com/pkg/errors"
)

// ErrTooLarge means yt-dlp refused the video because of --max-filesize.
var ErrTooLarge = errors.New("video is larger than the allowed size")

// ErrFiltered means yt-dlp refused the video because of --match-filter.
var ErrFiltered = errors.New("video does not pass the duration filter")

// classifyRejection maps yt-dlp output to a typed error when the run produced no
// file. yt-dlp exits successfully in both cases, so its output is the only signal.
func classifyRejection(output string) error {
	switch {
	case strings.Contains(output, "larger than max-filesize"):
		return ErrTooLarge
	case strings.Contains(output, "does not pass filter"):
		return ErrFiltered
	default:
		return nil
	}
}
