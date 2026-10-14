package grpc

import (
	"context"
	"errors"
	"fmt"
	"testing"

	projectsv1 "LogFlux/contracts/gen/go/projects/v1"
	"LogFlux/services/projects/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubValidator struct {
	err error
}

func (v stubValidator) ValidateKey(_ context.Context, _, _ string) (string, string, error) {
	if v.err != nil {
		return "", "", v.err
	}
	return "project-id", "key-id", nil
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name string
		req  *projectsv1.ValidateAPIKeyRequest
		err  error
		code codes.Code
	}{
		{name: "success", req: &projectsv1.ValidateAPIKeyRequest{}, code: codes.OK},
		{name: "missing request", code: codes.InvalidArgument},
		{name: "invalid input", req: &projectsv1.ValidateAPIKeyRequest{}, err: service.ErrInvalidAPIKeyInput, code: codes.InvalidArgument},
		{name: "invalid key", req: &projectsv1.ValidateAPIKeyRequest{}, err: fmt.Errorf("validate: %w", service.ErrInvalidAPIKey), code: codes.Unauthenticated},
		{name: "scope denied", req: &projectsv1.ValidateAPIKeyRequest{}, err: service.ErrAccessDenied, code: codes.PermissionDenied},
		{name: "canceled", req: &projectsv1.ValidateAPIKeyRequest{}, err: context.Canceled, code: codes.Canceled},
		{name: "deadline", req: &projectsv1.ValidateAPIKeyRequest{}, err: context.DeadlineExceeded, code: codes.DeadlineExceeded},
		{name: "unexpected failure", req: &projectsv1.ValidateAPIKeyRequest{}, err: errors.New("database unavailable"), code: codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewServer(stubValidator{err: tt.err})
			response, err := server.ValidateAPIKey(context.Background(), tt.req)
			if got := status.Code(err); got != tt.code {
				t.Fatalf("status code = %s, want %s", got, tt.code)
			}
			if tt.code == codes.OK {
				if response == nil || response.GetProjectId() != "project-id" || response.GetKeyId() != "key-id" {
					t.Fatalf("unexpected response: %+v", response)
				}
			} else if response != nil {
				t.Fatalf("response on error: %+v", response)
			}
		})
	}
}
