package middleware

import (
	"log/slog"
	"net/http"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "request", "method", r.Method, "url", r.URL)

		next.ServeHTTP(w, r)
	})
}
