package http

import (
	"errors"
	"net/http"
	"strings"
)

var ErrInvalidAuthorization = errors.New("invalid authorization")

func ExtractAPIKey(r *http.Request) (string, error) {
	values := r.Header.Values("Authorization")

	if len(values) != 1 {
		return "", ErrInvalidAuthorization
	}

	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidAuthorization
	}

	return values[1], nil
}

// TODO: валидация ключа через products service по GRPC
