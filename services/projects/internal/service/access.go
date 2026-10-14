package service

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"fmt"
	"uuid"
)

type AccessRepository interface {
	FindProjectRole(ctx context.Context, actorID, projectID uuid.UUID) (role model.TeamRole, found bool, err error)
}

type AccessService struct {
	repo AccessRepository
}

func NewAccessService(repo AccessRepository) *AccessService {
	return &AccessService{repo: repo}
}

func (s *AccessService) CanViewAPIKeys(ctx context.Context, actorID, projectID uuid.UUID) (bool, error) {
	role, found, err := s.repo.FindProjectRole(ctx, actorID, projectID)
	if err != nil {
		return false, fmt.Errorf("find role for project id %v: %w", projectID, err)
	}

	if !found {
		return false, nil
	}

	switch role {
	case model.TeamRoleMember, model.TeamRoleOwner, model.TeamRoleViewer:
		return true, nil
	default:
		return false, nil
	}
}
func (s *AccessService) CanManageAPIKeys(ctx context.Context, actorID, projectID uuid.UUID) (bool, error) {
	role, found, err := s.repo.FindProjectRole(ctx, actorID, projectID)
	if err != nil {
		return false, fmt.Errorf("find project role: %w", err)
	}

	return found && role == model.TeamRoleOwner, nil
}
