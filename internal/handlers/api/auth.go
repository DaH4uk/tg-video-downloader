package api

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"tg-video-downloader/internal/infrastructure/metrics"
)

const bearerPrefix = "Bearer "

// BearerAuth rejects every request that does not carry the exact API token.
func BearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			unauthorized(w)
			return
		}

		got := strings.TrimPrefix(header, bearerPrefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// unauthorized writes the 401 response and records it, mirroring
// DownloadHandler.fail so unauthorized attempts show up in tgvd_api_requests_total
// alongside every other response status.
func unauthorized(w http.ResponseWriter) {
	metrics.APIRequests.WithLabelValues(strconv.Itoa(http.StatusUnauthorized)).Inc()
	WriteError(w, http.StatusUnauthorized, "unauthorized")
}
