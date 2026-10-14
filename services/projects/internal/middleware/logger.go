package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	status  int
	bytes   int64
	aborted bool
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += int64(n)
	return n, err
}

func Logger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &responseRecorder{ResponseWriter: w}
		logger.LogAttrs(r.Context(), slog.LevelInfo, "http_request_started",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)

		defer func() {
			status := recorder.status
			if status == 0 && !recorder.aborted {
				status = http.StatusOK
			}

			level := slog.LevelInfo
			if recorder.aborted || status >= 500 {
				level = slog.LevelError
			} else if status >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, "http_request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Bool("aborted", recorder.aborted),
				slog.Duration("duration", time.Since(started)),
				slog.Int64("response_bytes", recorder.bytes),
			)
		}()

		next.ServeHTTP(recorder, r)
	})
}
