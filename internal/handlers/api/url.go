package api

import (
	"net"
	"net/url"

	"github.com/pkg/errors"
)

// ErrInvalidURL means the request URL is malformed or does not use https.
var ErrInvalidURL = errors.New("url must be a valid https:// address")

// ErrForbiddenHost means the URL points at a loopback, private or link-local address.
var ErrForbiddenHost = errors.New("url host is not allowed")

// ValidateVideoURL checks the URL before it is handed to yt-dlp. DNS is not
// resolved on purpose: a domain pointing at 127.0.0.1 passes this check. See the
// design doc for why that trade-off is acceptable here.
func ValidateVideoURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}

	if parsed.Scheme != "https" || parsed.Host == "" {
		return ErrInvalidURL
	}

	host := parsed.Hostname()
	if host == "" {
		return ErrInvalidURL
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return ErrForbiddenHost
	}

	return nil
}
