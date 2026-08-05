package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const bearerPrefix = "Bearer "

// BearerAuth rejects every request that does not carry the exact API token.
func BearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, bearerPrefix) {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		got := strings.TrimPrefix(header, bearerPrefix)
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
