package service

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"uuid"
)

type UserRepository interface {
}

type AuthService struct {
	repo UserRepository
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {

}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {

}

func (s *AuthService) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {

}

func (s *AuthService) FindUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {

}
