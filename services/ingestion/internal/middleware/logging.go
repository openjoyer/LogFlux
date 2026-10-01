package middleware

import (
	"log/slog"
	"net/http"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("%s %s", r.Method, r.URL)

		next.ServeHTTP(w, r)
	})
}
