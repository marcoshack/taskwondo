package middleware

import (
	"net/http"
	"strings"
)

// BodyLimit restricts the size of request bodies to maxBytes.
// Multipart form-data requests (file uploads) are exempt since they
// have their own size limits applied at the handler level.
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if r.Body != nil && !strings.HasPrefix(ct, "multipart/form-data") {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
