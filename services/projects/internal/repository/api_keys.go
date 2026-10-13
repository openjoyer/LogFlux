package repository

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepository struct {
	pool *pgxpool.Pool
}

func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) FindAPIKey(ctx context.Context, keyID string) (*model.APIKey, error) {
	var key model.APIKey
	var expiresAt, revokedAt pgtype.Timestamptz

	err := r.pool.QueryRow(
		ctx, `SELECT 
    	id,
       	project_id,
      	name, 
       	key_hash, 
       	scopes,
		created_at, 
		expires_at, 
		revoked_at
		FROM api_keys WHERE id = $1
	`, keyID).Scan(
		&key.ID,
		&key.ProjectID,
		&key.Name,
		&key.KeyHash,
		&key.Scopes,
		&key.CreatedAt,
		&key.ExpiresAt,
		&key.RevokedAt)
	if err != nil {
		return nil, fmt.Errorf("find api key: %w", err)
	}

	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		key.RevokedAt = &revokedAt.Time
	}

	return &key, nil
}

func (r *APIKeyRepository) CreateAPIKey(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	if key == nil {
		return nil, errors.New("create API key: nil key")
	}

	created := *key
	created.RevokedAt = nil

	err := r.pool.QueryRow(ctx, `
		INSERT INTO api_keys (
			project_id, name, key_hash, scopes, expires_at
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`,
		key.ProjectID,
		key.Name,
		key.KeyHash,
		key.Scopes,
		key.ExpiresAt,
	).Scan(&created.ID, &created.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create API key: %w", err)
	}
	return &created, nil
}

func (r *APIKeyRepository) FindAllAPIKeys(ctx context.Context, projectID uuid.UUID) ([]model.APIKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, scopes,
		       created_at, expires_at, revoked_at
		FROM api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC, id DESC
	`, projectID)

	if err != nil {
		return nil, fmt.Errorf("find all API keys: %w", err)
	}
	defer rows.Close()

	keys := make([]model.APIKey, 0)
	for rows.Next() {
		var key model.APIKey
		var expiresAt, revokedAt pgtype.Timestamptz

		err := rows.Scan(
			&key.ID,
			&key.ProjectID,
			&key.Name,
			&key.Scopes,
			&key.CreatedAt,
			&expiresAt,
			&revokedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("find all API keys: %w", err)
		}

		if expiresAt.Valid {
			value := expiresAt.Time
			key.ExpiresAt = &value
		}
		if revokedAt.Valid {
			value := revokedAt.Time
			key.RevokedAt = &value
		}

		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate API keys: %w", err)
	}

	return keys, nil
}

func (r *APIKeyRepository) UpdateAPIKey(ctx context.Context, key *model.APIKey) (*model.APIKey, error) {
	if key == nil {
		return nil, errors.New("update API key: nil key")
	}

	updated, err := scanAPIKey(r.pool.QueryRow(ctx, `
		UPDATE api_keys
		SET name = $3,
		    scopes = $4,
		    expires_at = $5
		WHERE id = $1
		  AND project_id = $2
		  AND revoked_at IS NULL
		RETURNING id, project_id, name, key_hash, scopes,
		          created_at, expires_at, revoked_at
	`,
		key.ID,
		key.ProjectID,
		key.Name,
		key.Scopes,
		key.ExpiresAt,
	))

	if err != nil {
		return nil, fmt.Errorf("update API key: %w", err)
	}

	return updated, nil
}

func (r *APIKeyRepository) RevokeAPIKey(ctx context.Context, projectID, keyID uuid.UUID, revokedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE api_keys
		SET revoked_at = $3
		WHERE project_id = $1
		  AND id = $2
		  AND revoked_at IS NULL
	`, projectID, keyID, revokedAt)
	if err != nil {
		return fmt.Errorf("revoke API key: %w", err)
	}

	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(row rowScanner) (*model.APIKey, error) {
	var key model.APIKey
	var expiresAt, revokedAt pgtype.Timestamptz

	err := row.Scan(
		&key.ID,
		&key.ProjectID,
		&key.Name,
		&key.KeyHash,
		&key.Scopes,
		&key.CreatedAt,
		&expiresAt,
		&revokedAt,
	)
	if err != nil {
		return nil, err
	}

	if expiresAt.Valid {
		value := expiresAt.Time
		key.ExpiresAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time
		key.RevokedAt = &value
	}

	return &key, nil
}
