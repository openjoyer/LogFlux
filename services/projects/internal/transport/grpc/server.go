package grpc

import (
	projectsv1 "LogFlux/contracts/gen/go/projects/v1"
	"context"
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
	projectID, keyID, err := s.validator.ValidateKey(ctx, req.Key, req.RequiredScope)

	if err != nil {
		return nil, err
	}

	return &projectsv1.ValidateAPIKeyResponse{
		KeyId:     keyID,
		ProjectId: projectID,
	}, nil
}
