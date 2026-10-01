package http

import "net/http"

func NewRouter(
	h *Handler,
	common func(http.Handler) http.Handler,
	protected func(http.Handler) http.Handler,
) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health/live", h.CheckLiveness)

	mux.Handle("POST /api/logs", protected(http.HandlerFunc(h.UploadLogs)))

	return common(mux)
}
