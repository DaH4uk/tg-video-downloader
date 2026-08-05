package api

import (
	"fmt"
	"net/url"
	"strings"
)

// contentDisposition builds a header value that survives non-ASCII titles: an
// ASCII fallback for the filename parameter and an RFC 5987 form with the real
// name. Video titles routinely contain Cyrillic and emoji.
func contentDisposition(filename string) string {
	var ascii strings.Builder
	for _, r := range filename {
		if r < 128 && r != '"' && r != '\\' {
			ascii.WriteRune(r)
			continue
		}
		ascii.WriteByte('_')
	}

	fallback := ascii.String()
	if strings.Trim(fallback, "_") == "" {
		fallback = "video.mp4"
	}

	return fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", fallback, url.PathEscape(filename))
}
