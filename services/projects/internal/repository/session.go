package repository

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	if session == nil {
		return errors.New("create session: nil session")
	}
	if len(session.TokenHash) != sha256.Size {
		return errors.New("create session: invalid token hash length")
	}
	if session.ExpiresAt.IsZero() {
		return errors.New("create session: missing expiration")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, session.UserID, session.TokenHash, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert user session: %w", err)
	}
	return nil
}

func (r *SessionRepository) Find(ctx context.Context, tokenHash []byte) (*model.Session, error) {
	if len(tokenHash) != sha256.Size {
		return nil, nil
	}

	var session model.Session
	err := r.pool.QueryRow(ctx, `
		SELECT s.user_id, s.token_hash, s.expires_at
		FROM user_sessions AS s
		JOIN users AS u ON u.id = s.user_id
		WHERE s.token_hash = $1
	`, tokenHash).Scan(&session.UserID, &session.TokenHash, &session.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user session: %w", err)
	}
	return &session, nil
}
