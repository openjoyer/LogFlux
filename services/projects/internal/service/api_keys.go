package service

import (
	"LogFlux/services/projects/internal/api_key"
	"LogFlux/services/projects/internal/model"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"uuid"
)

const ScopeLogsWrite = "logs:write"

type APIKeyRepository interface {
	FindAPIKey(ctx context.Context, keyID string) (*model.APIKey, error)
	CreateAPIKey(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
	FindAllAPIKeys(ctx context.Context, projectID uuid.UUID) ([]model.APIKey, error)
	UpdateAPIKey(ctx context.Context, key *model.APIKey) (*model.APIKey, error)
	RevokeAPIKey(ctx context.Context, projectID, keyID uuid.UUID, revokedAt time.Time) error
}

type APIKeyAccess interface {
	CanViewAPIKeys(ctx context.Context, actorID, projectID uuid.UUID) (bool, error)
	CanManageAPIKeys(ctx context.Context, actorID, projectID uuid.UUID) (bool, error)
}

type APIKeyService struct {
	repo   APIKeyRepository
	access APIKeyAccess
}

func NewAPIKeyValidator(repo APIKeyRepository, access APIKeyAccess) *APIKeyService {
	return &APIKeyService{repo: repo, access: access}
}

type apiKeyError struct {
	code    string
	message string
}

func (e *apiKeyError) Error() string { return e.message }
func (e *apiKeyError) Code() string  { return e.code }

var (
	ErrInvalidAPIKey = &apiKeyError{
		code:    "unauthenticated",
		message: "invalid API key",
	}
	ErrInvalidAPIKeyInput = &apiKeyError{
		code:    "invalid_argument",
		message: "invalid API key input",
	}
	ErrAccessDenied = &apiKeyError{
		code:    "permission_denied",
		message: "access denied",
	}
)

func (s *APIKeyService) Create(ctx context.Context, actorID, projectID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (*model.APIKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", ErrInvalidAPIKeyInput
	}

	normalizedScopes, err := normalizeScopes(scopes)
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	var expiration *time.Time
	if expiresAt != nil {
		value := expiresAt.UTC()
		if !value.After(now) {
			return nil, "", ErrInvalidAPIKeyInput
		}
		expiration = &value
	}

	allowed, err := s.access.CanManageAPIKeys(ctx, actorID, projectID)
	if err != nil {
		return nil, "", fmt.Errorf("check API key management access: %w", err)
	}
	if !allowed {
		return nil, "", ErrAccessDenied
	}

	secret, keyHash, err := api_key.GenerateSecret()
	if err != nil {
		return nil, "", fmt.Errorf("generate API key secret: %w", err)
	}

	key := &model.APIKey{
		ProjectID: projectID,
		Name:      name,
		KeyHash:   keyHash,
		Scopes:    normalizedScopes,
		ExpiresAt: expiration,
	}

	// Репозиторий должен записать ключ и вернуть сгенерированный БД ID.
	saved, err := s.repo.CreateAPIKey(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("create API key: %w", err)
	}
	if saved == nil {
		return nil, "", errors.New("create API key: repository returned nil")
	}

	rawKey := "lfk_" + saved.ID.String() + "." + secret

	publicKey := *saved
	publicKey.KeyHash = nil

	return &publicKey, rawKey, nil
}

func (s *APIKeyService) FindAll(ctx context.Context, actorID, projectID uuid.UUID) ([]model.APIKey, error) {
	allowed, err := s.access.CanViewAPIKeys(ctx, actorID, projectID)
	if err != nil {
		return nil, fmt.Errorf("check API key view access: %w", err)
	}
	if !allowed {
		return nil, ErrAccessDenied
	}

	keys, err := s.repo.FindAllAPIKeys(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("find project API keys: %w", err)
	}

	result := make([]model.APIKey, len(keys))
	for i := range keys {
		result[i] = keys[i]
		result[i].KeyHash = nil
	}

	return result, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, actorID, projectID, keyID uuid.UUID) error {
	allowed, err := s.access.CanManageAPIKeys(ctx, actorID, projectID)
	if err != nil {
		return fmt.Errorf("check API key management access: %w", err)
	}
	if !allowed {
		return ErrAccessDenied
	}

	if err := s.repo.RevokeAPIKey(
		ctx,
		projectID,
		keyID,
		time.Now().UTC(),
	); err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}

	return nil
}

func (s *APIKeyService) ValidateKey(
	ctx context.Context,
	rawKey string,
	requiredScope string) (string, string, error) {
	if requiredScope == "" {
		return "", "", ErrInvalidAPIKeyInput
	}

	parsedKeyID, secret, err := s.parseAPIKey(rawKey)
	if err != nil {
		return "", "", ErrInvalidAPIKey
	}

	key, err := s.repo.FindAPIKey(ctx, parsedKeyID)
	if err != nil {
		return "", "", fmt.Errorf("find API key: %w", err)
	}
	if key == nil {
		return "", "", ErrInvalidAPIKey
	}

	if key.ID.String() != parsedKeyID ||
		!api_key.VerifySecret(secret, key.KeyHash) {
		return "", "", ErrInvalidAPIKey
	}

	if key.RevokedAt != nil {
		return "", "", ErrInvalidAPIKey
	}
	if key.ExpiresAt != nil && !time.Now().Before(*key.ExpiresAt) {
		return "", "", ErrInvalidAPIKey
	}

	if !slices.Contains(key.Scopes, requiredScope) {
		return "", "", ErrAccessDenied
	}

	return key.ProjectID.String(), key.ID.String(), nil
}

// Bearer lfk_<id>.<secret>
func (s *APIKeyService) parseAPIKey(raw string) (keyID string, secret string, err error) {
	rest, ok := strings.CutPrefix(raw, "lfk_")
	if !ok {
		return "", "", ErrInvalidAPIKey
	}

	idText, secret, ok := strings.Cut(rest, ".")
	if !ok || secret == "" {
		return "", "", ErrInvalidAPIKey
	}

	id, err := uuid.Parse(idText)
	if err != nil || id.String() != idText {
		return "", "", ErrInvalidAPIKey
	}

	return id.String(), secret, nil
}

func (s *APIKeyService) validateSecret(key *model.APIKey, encodedSecret string) bool {
	return api_key.VerifySecret(encodedSecret, key.KeyHash)
}

func (s *APIKeyService) validateScope(key *model.APIKey, scope string) bool {
	return slices.Contains(key.Scopes, scope)
}

func normalizeScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return nil, ErrInvalidAPIKeyInput
	}

	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)

		switch scope {
		case ScopeLogsWrite:
			if !slices.Contains(result, scope) {
				result = append(result, scope)
			}
		default:
			return nil, ErrInvalidAPIKeyInput
		}
	}

	return result, nil
}
