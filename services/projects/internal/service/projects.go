package service

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"uuid"
)

type ProjectRepository interface {
}

type ProjectService struct {
	repo ProjectRepository
}

func NewProjectService(repo ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) FindAll(ctx context.Context, actorID, teamID uuid.UUID) ([]model.Project, error) {

}

func (s *ProjectService) Create(ctx context.Context, actorID, teamID uuid.UUID, name string) (*model.Project, error) {

}

func (s *ProjectService) FindByID(ctx context.Context, actorID, projectID uuid.UUID) (*model.Project, error) {

}

func (s *ProjectService) Update(ctx context.Context, actorID, projectID uuid.UUID, name *string, status *model.ProjectStatus) (*model.Project, error) {

}
