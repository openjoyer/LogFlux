package middleware

import (
	. "LogFlux/services/ingestion/internal/transport/http"
	"context"
	"net/http"
)

type Limiter interface {
	Allow(ctx context.Context, projectID string) (bool, error)
}

func RateLimiting(limiter Limiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, err := ExtractAPIKey(r) // TODO передавать проверенный projectID

			if err != nil {
				http.Error(w, "missing or invalid API key", http.StatusUnauthorized)
				return
			}

			ok, err := limiter.Allow(r.Context(), key)
			if err != nil {
				http.Error(w, "rate limiter: unavailable", http.StatusServiceUnavailable)
				return
			}

			if !ok {
				http.Error(w, "rate limiter: too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
