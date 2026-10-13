package service

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"uuid"
)

type TeamRepository interface {

}

type TeamService struct {
	repo TeamRepository
}

func (s *TeamService) FindAll(ctx context.Context, actorID uuid.UUID) ([]model.Team, error) {

}

func (s *TeamService) Create(ctx context.Context, actorID uuid.UUID, name string) (*model.Team, error) {

}

func (s *TeamService) FindMembers(ctx context.Context, actorID, teamID uuid.UUID) ([]model.TeamMemberShip, error) {

}

func (s *TeamService) AddMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {

}

func (s *TeamService) UpdateMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {

}

func (s *TeamService) DeleteMember(ctx context.Context, actorID, teamID, userID uuid.UUID) error {

}