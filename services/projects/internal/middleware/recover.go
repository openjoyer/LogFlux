package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recover(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder, ok := w.(*responseRecorder)
		if !ok {
			recorder = &responseRecorder{ResponseWriter: w}
		}

		defer func() {
			value := recover()
			if value == nil {
				return
			}
			if value == http.ErrAbortHandler {
				recorder.aborted = true
				panic(value)
			}

			logger.ErrorContext(r.Context(), "http_panic",
				"method", r.Method,
				"path", r.URL.Path,
				"panic", fmt.Sprint(value),
				"stack", string(debug.Stack()),
			)

			if recorder.status != 0 {
				recorder.aborted = true
				panic(http.ErrAbortHandler)
			}

			recorder.Header().Set("Content-Type", "application/json")
			recorder.WriteHeader(http.StatusInternalServerError)
			_, _ = recorder.Write([]byte("{\"error\":\"internal server error\"}\n"))
		}()

		next.ServeHTTP(recorder, r)
	})
}
