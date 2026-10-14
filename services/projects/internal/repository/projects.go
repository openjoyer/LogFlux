package repository

import (
	"context"
	"errors"
	"fmt"

	"LogFlux/services/projects/internal/model"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) FindAll(ctx context.Context, teamID uuid.UUID) ([]model.Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, team_id, name, status, created_at, updated_at
		FROM projects
		WHERE team_id = $1
		ORDER BY created_at DESC, id DESC
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query team projects: %w", err)
	}
	defer rows.Close()

	projects := make([]model.Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("scan team project: %w", err)
		}
		projects = append(projects, *project)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team projects: %w", err)
	}
	return projects, nil
}

func (r *ProjectRepository) Create(ctx context.Context, actorID uuid.UUID, project *model.Project) (*model.Project, error) {
	if project == nil {
		return nil, errors.New("create project: nil project")
	}

	created, err := scanProject(r.pool.QueryRow(ctx, `
		INSERT INTO projects (team_id, name, status)
		SELECT $1, $2, $3
		WHERE EXISTS (
		    SELECT 1
		    FROM team_memberships
		    WHERE team_id = $1
		      AND user_id = $4
		      AND role = $5
		)
		RETURNING id, team_id, name, status, created_at, updated_at
	`, project.TeamID, project.Name, project.Status, actorID, model.TeamRoleOwner))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return created, nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, projectID uuid.UUID) (*model.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, `
		SELECT id, team_id, name, status, created_at, updated_at
		FROM projects
		WHERE id = $1
	`, projectID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	return project, nil
}

func (r *ProjectRepository) Update(
	ctx context.Context,
	actorID, projectID uuid.UUID,
	name *string,
	status *model.ProjectStatus,
) (*model.Project, error) {
	project, err := scanProject(r.pool.QueryRow(ctx, `
		UPDATE projects AS p
		SET name = COALESCE($2, name),
		    status = COALESCE($3, status),
		    updated_at = now()
		WHERE p.id = $1
		  AND EXISTS (
		      SELECT 1
		      FROM team_memberships AS tm
		      WHERE tm.team_id = p.team_id
		        AND tm.user_id = $4
		        AND tm.role = $5
		  )
		RETURNING id, team_id, name, status, created_at, updated_at
	`, projectID, name, status, actorID, model.TeamRoleOwner))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return project, nil
}

type projectRowScanner interface {
	Scan(dest ...any) error
}

func scanProject(row projectRowScanner) (*model.Project, error) {
	var project model.Project
	err := row.Scan(
		&project.ID,
		&project.TeamID,
		&project.Name,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &project, nil
}
