package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"LogFlux/services/projects/internal/model"
	"uuid"
)

type TeamRepository interface {
	FindAll(ctx context.Context, actorID uuid.UUID) ([]model.Team, error)
	Create(ctx context.Context, actorID uuid.UUID, name string) (*model.Team, error)
	FindMembers(ctx context.Context, teamID uuid.UUID) ([]model.TeamMemberShip, error)
	AddMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error)
	UpdateMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error)
	DeleteMember(ctx context.Context, actorID, teamID, userID uuid.UUID) (deleted bool, err error)
}

type TeamAccess interface {
	FindTeamRole(ctx context.Context, actorID, teamID uuid.UUID) (role model.TeamRole, found bool, err error)
}

type TeamService struct {
	repo   TeamRepository
	access TeamAccess
}

func NewTeamService(repo TeamRepository, access TeamAccess) *TeamService {
	return &TeamService{repo: repo, access: access}
}

type teamError struct {
	code    string
	message string
}

func (e *teamError) Error() string { return e.message }
func (e *teamError) Code() string  { return e.code }

var (
	ErrInvalidTeamInput   = &teamError{code: "invalid_argument", message: "invalid team input"}
	ErrTeamAccessDenied   = &teamError{code: "permission_denied", message: "access denied"}
	ErrTeamNotFound       = &teamError{code: "not_found", message: "team not found"}
	ErrTeamMemberNotFound = &teamError{code: "not_found", message: "team member not found"}
)

func (s *TeamService) FindAll(ctx context.Context, actorID uuid.UUID) ([]model.Team, error) {
	teams, err := s.repo.FindAll(ctx, actorID)
	if err != nil {
		return nil, fmt.Errorf("find user teams: %w", err)
	}
	if teams == nil {
		return []model.Team{}, nil
	}
	return teams, nil
}

func (s *TeamService) Create(ctx context.Context, actorID uuid.UUID, name string) (*model.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidTeamInput
	}

	team, err := s.repo.Create(ctx, actorID, name)
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	if team == nil {
		return nil, errors.New("create team: repository returned nil")
	}
	return team, nil
}

func (s *TeamService) FindMembers(ctx context.Context, actorID, teamID uuid.UUID) ([]model.TeamMemberShip, error) {
	role, found, err := s.access.FindTeamRole(ctx, actorID, teamID)
	if err != nil {
		return nil, fmt.Errorf("check team membership: %w", err)
	}
	if !found || !validTeamRole(role) {
		return nil, ErrTeamNotFound
	}

	members, err := s.repo.FindMembers(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("find team members: %w", err)
	}
	if members == nil {
		return []model.TeamMemberShip{}, nil
	}
	return members, nil
}

func (s *TeamService) AddMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {
	if !validTeamRole(role) {
		return nil, ErrInvalidTeamInput
	}
	if err := s.requireOwner(ctx, actorID, teamID); err != nil {
		return nil, err
	}

	member, err := s.repo.AddMember(ctx, actorID, teamID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("add team member: %w", err)
	}
	if member == nil {
		return nil, ErrTeamNotFound
	}
	return member, nil
}

func (s *TeamService) UpdateMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {
	if !validTeamRole(role) {
		return nil, ErrInvalidTeamInput
	}
	if err := s.requireOwner(ctx, actorID, teamID); err != nil {
		return nil, err
	}

	member, err := s.repo.UpdateMember(ctx, actorID, teamID, userID, role)
	if err != nil {
		return nil, fmt.Errorf("update team member: %w", err)
	}
	if member == nil {
		return nil, ErrTeamMemberNotFound
	}
	return member, nil
}

func (s *TeamService) DeleteMember(ctx context.Context, actorID, teamID, userID uuid.UUID) error {
	if err := s.requireOwner(ctx, actorID, teamID); err != nil {
		return err
	}

	deleted, err := s.repo.DeleteMember(ctx, actorID, teamID, userID)
	if err != nil {
		return fmt.Errorf("delete team member: %w", err)
	}
	if !deleted {
		return ErrTeamMemberNotFound
	}
	return nil
}

func (s *TeamService) requireOwner(ctx context.Context, actorID, teamID uuid.UUID) error {
	role, found, err := s.access.FindTeamRole(ctx, actorID, teamID)
	if err != nil {
		return fmt.Errorf("check team ownership: %w", err)
	}
	if !found || role != model.TeamRoleOwner {
		return ErrTeamAccessDenied
	}
	return nil
}

func validTeamRole(role model.TeamRole) bool {
	switch role {
	case model.TeamRoleOwner, model.TeamRoleMember, model.TeamRoleViewer:
		return true
	default:
		return false
	}
}
