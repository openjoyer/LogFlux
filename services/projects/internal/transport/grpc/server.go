package grpc

import (
	"context"
	"errors"

	projectsv1 "LogFlux/contracts/gen/go/projects/v1"
	"LogFlux/services/projects/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type APIKeyValidator interface {
	ValidateKey(ctx context.Context, rawKey string, requiredScope string) (projectID string, keyID string, err error)
}

type Server struct {
	projectsv1.UnimplementedProjectAuthServiceServer
	validator APIKeyValidator
}

func NewServer(validator APIKeyValidator) *Server {
	return &Server{validator: validator}
}

func (s *Server) ValidateAPIKey(
	ctx context.Context,
	req *projectsv1.ValidateAPIKeyRequest,
) (*projectsv1.ValidateAPIKeyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	projectID, keyID, err := s.validator.ValidateKey(ctx, req.Key, req.RequiredScope)

	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidAPIKeyInput):
			return nil, status.Error(codes.InvalidArgument, "invalid API key input")
		case errors.Is(err, service.ErrInvalidAPIKey):
			return nil, status.Error(codes.Unauthenticated, "invalid API key")
		case errors.Is(err, service.ErrAccessDenied):
			return nil, status.Error(codes.PermissionDenied, "access denied")
		case errors.Is(err, context.Canceled):
			return nil, status.Error(codes.Canceled, "request canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return nil, status.Error(codes.DeadlineExceeded, "deadline exceeded")
		default:
			return nil, status.Error(codes.Internal, "internal server error")
		}
	}

	return &projectsv1.ValidateAPIKeyResponse{
		KeyId:     keyID,
		ProjectId: projectID,
	}, nil
}
