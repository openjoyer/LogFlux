package service

import (
	"context"
	"fmt"
	"strings"

	"LogFlux/services/projects/internal/model"
	"uuid"
)

type ProjectRepository interface {
	FindAll(ctx context.Context, teamID uuid.UUID) ([]model.Project, error)
	Create(ctx context.Context, actorID uuid.UUID, project *model.Project) (*model.Project, error)
	FindByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error)
	Update(ctx context.Context, actorID, projectID uuid.UUID, name *string, status *model.ProjectStatus) (*model.Project, error)
}

type ProjectAccess interface {
	FindTeamRole(ctx context.Context, actorID, teamID uuid.UUID) (role model.TeamRole, found bool, err error)
	FindProjectRole(ctx context.Context, actorID, projectID uuid.UUID) (role model.TeamRole, found bool, err error)
}

type ProjectService struct {
	repo   ProjectRepository
	access ProjectAccess
}

func NewProjectService(repo ProjectRepository, access ProjectAccess) *ProjectService {
	return &ProjectService{repo: repo, access: access}
}

type projectError struct {
	code    string
	message string
}

func (e *projectError) Error() string { return e.message }
func (e *projectError) Code() string  { return e.code }

var (
	ErrProjectNotFound = &projectError{
		code:    "not_found",
		message: "project not found",
	}
	ErrInvalidProjectInput = &projectError{
		code:    "invalid_argument",
		message: "invalid project input",
	}
	ErrProjectAccessDenied = &projectError{
		code:    "permission_denied",
		message: "access denied",
	}
)

func (s *ProjectService) FindAll(ctx context.Context, actorID, teamID uuid.UUID) ([]model.Project, error) {
	role, found, err := s.access.FindTeamRole(ctx, actorID, teamID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if !found || !canViewProjects(role) {
		return nil, ErrProjectAccessDenied
	}

	projects, err := s.repo.FindAll(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("find team projects: %w", err)
	}
	if projects == nil {
		return []model.Project{}, nil
	}
	return projects, nil
}

func (s *ProjectService) Create(ctx context.Context, actorID, teamID uuid.UUID, name string) (*model.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidProjectInput
	}

	role, found, err := s.access.FindTeamRole(ctx, actorID, teamID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if !found || role != model.TeamRoleOwner {
		return nil, ErrProjectAccessDenied
	}

	project, err := s.repo.Create(ctx, actorID, &model.Project{
		TeamID: teamID,
		Name:   name,
		Status: model.ProjectStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	if project == nil {
		return nil, ErrProjectAccessDenied
	}
	return project, nil
}

func (s *ProjectService) FindByID(ctx context.Context, actorID, projectID uuid.UUID) (*model.Project, error) {
	role, found, err := s.access.FindProjectRole(ctx, actorID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project membership: %w", err)
	}
	if !found || !canViewProjects(role) {
		return nil, ErrProjectNotFound
	}

	project, err := s.repo.FindByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

func (s *ProjectService) Update(ctx context.Context, actorID, projectID uuid.UUID, name *string, status *model.ProjectStatus) (*model.Project, error) {
	if name == nil && status == nil {
		return nil, ErrInvalidProjectInput
	}
	if name != nil {
		trimmed := strings.TrimSpace(*name)
		if trimmed == "" {
			return nil, ErrInvalidProjectInput
		}
		name = &trimmed
	}
	if status != nil &&
		*status != model.ProjectStatusActive &&
		*status != model.ProjectStatusArchived {
		return nil, ErrInvalidProjectInput
	}

	role, found, err := s.access.FindProjectRole(ctx, actorID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check project membership: %w", err)
	}
	if !found || role != model.TeamRoleOwner {
		return nil, ErrProjectNotFound
	}

	project, err := s.repo.Update(ctx, actorID, projectID, name, status)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}
	return project, nil
}

func canViewProjects(role model.TeamRole) bool {
	switch role {
	case model.TeamRoleOwner, model.TeamRoleMember, model.TeamRoleViewer:
		return true
	default:
		return false
	}
}
