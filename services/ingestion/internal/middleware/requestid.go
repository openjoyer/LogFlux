package middleware

import (
	"net/http"
	"uuid"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()

		r.Header.Set("X-Request-Id", id)
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r)
	})
}
