package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging logs each HTTP request with method, path, status, and duration.
// It also injects the logger into the request context for downstream use via log.Ctx(ctx).
func Logging(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := GetRequestID(r.Context())
			reqLogger := logger.With().Str("request_id", requestID).Logger()
			ctx := reqLogger.WithContext(r.Context())

			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r.WithContext(ctx))

			log.Ctx(ctx).Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.statusCode).
				Int64("duration_ms", time.Since(start).Milliseconds()).
				Str("remote_addr", r.RemoteAddr).
				Msg("http request")
		})
	}
}
