package repository

import (
	"LogFlux/services/projects/internal/model"
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AccessRepository struct {
	pool *pgxpool.Pool
}

func NewAccessRepository(pool *pgxpool.Pool) *AccessRepository {
	return &AccessRepository{pool: pool}
}

func (r *AccessRepository) FindProjectRole(ctx context.Context, actorID, projectID uuid.UUID) (model.TeamRole, bool, error) {
	var role model.TeamRole
	err := r.pool.QueryRow(ctx, `
		SELECT tm.role
		FROM projects AS p
		JOIN team_memberships AS tm ON tm.team_id = p.team_id
		WHERE p.id = $1 AND tm.user_id = $2
	`, projectID, actorID).Scan(&role)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("find project membership: %w", err)
	}

	return role, true, nil
}
