package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"LogFlux/services/projects/internal/model"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, userID uuid.UUID) (*model.User, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	Find(ctx context.Context, tokenHash []byte) (*model.Session, error)
}
type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	sessionTTL  time.Duration
	dummyHash   []byte
}

const (
	userTokenPrefix = "lfu_"
	userTokenSize   = 32
	minPasswordSize = 8
)

type authError struct {
	code    string
	message string
}

func (e *authError) Error() string { return e.message }
func (e *authError) Code() string  { return e.code }

var (
	ErrInvalidAuthInput   = &authError{code: "invalid_argument", message: "invalid authentication input"}
	ErrInvalidCredentials = &authError{code: "unauthenticated", message: "invalid credentials"}
	ErrInvalidSession     = &authError{code: "unauthenticated", message: "invalid session"}
	ErrUserNotFound       = &authError{code: "not_found", message: "user not found"}
)

func NewAuthService(userRepo UserRepository, sessionRepo SessionRepository, sessionTTL time.Duration) (*AuthService, error) {
	if sessionTTL <= 0 {
		return nil, errors.New("session TTL must be positive")
	}
	dummyHash, err := bcrypt.GenerateFromPassword([]byte("unused-account-password"), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("prepare password verification: %w", err)
	}
	return &AuthService{userRepo: userRepo, sessionRepo: sessionRepo, sessionTTL: sessionTTL, dummyHash: dummyHash}, nil
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.User, error) {
	email, err := normalizeEmail(email)
	if err != nil || len(password) < minPasswordSize || len(password) > 72 {
		return nil, ErrInvalidAuthInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.userRepo.Create(ctx, email, string(hash))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	if user == nil {
		return nil, errors.New("create user: repository returned nil")
	}
	return publicUser(user), nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	email, err := normalizeEmail(email)
	if err != nil || password == "" {
		return "", ErrInvalidCredentials
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("find user by email: %w", err)
	}
	if user == nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("verify password: %w", err)
	}

	secret := make([]byte, userTokenSize)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	tokenHash := sha256.Sum256(secret)
	session := &model.Session{
		UserID:    user.ID,
		TokenHash: tokenHash[:],
		ExpiresAt: time.Now().UTC().Add(s.sessionTTL),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", fmt.Errorf("create user session: %w", err)
	}

	return userTokenPrefix + base64.RawURLEncoding.EncodeToString(secret), nil
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (uuid.UUID, error) {
	encoded, ok := strings.CutPrefix(token, userTokenPrefix)
	if !ok || len(encoded) != base64.RawURLEncoding.EncodedLen(userTokenSize) {
		return uuid.UUID{}, ErrInvalidSession
	}
	secret, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(secret) != userTokenSize {
		return uuid.UUID{}, ErrInvalidSession
	}

	tokenHash := sha256.Sum256(secret)
	session, err := s.sessionRepo.Find(ctx, tokenHash[:])
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("find user session: %w", err)
	}
	if session == nil || !time.Now().Before(session.ExpiresAt) {
		return uuid.UUID{}, ErrInvalidSession
	}
	return session.UserID, nil
}

func (s *AuthService) FindUser(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return publicUser(user), nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) == 0 || len(email) > 254 {
		return "", ErrInvalidAuthInput
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", ErrInvalidAuthInput
	}
	return email, nil
}

func publicUser(user *model.User) *model.User {
	copy := *user
	copy.PasswordHash = ""
	return &copy
}
