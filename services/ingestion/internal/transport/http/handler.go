package http

import (
	"context"
	"net/http"
)

type IngestionService interface {
	UploadLogs(ctx context.Context, logs []Log) error
}

type Handler struct {
	IngestionService IngestionService
}

func NewHandler(s IngestionService) *Handler {
	return &Handler{s}
}

func (h *Handler) UploadLogs(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) CheckLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
