package middleware

import (
	httptransport "LogFlux/services/ingestion/internal/transport/http"
	"fmt"
	"net/http"
)

func Authentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, err := httptransport.ExtractAPIKey(r)

		if err != nil {
			fmt.Println(key)
		}
		// TODO валидация через GRPC в products service
		next.ServeHTTP(w, r)
	})
}
