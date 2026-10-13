package repository

import (
	"context"
	"errors"
	"fmt"

	"LogFlux/services/projects/internal/model"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeamRepository struct {
	pool *pgxpool.Pool
}

func NewTeamRepository(pool *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{pool: pool}
}

type teamRepositoryError struct {
	code    string
	message string
}

func (e *teamRepositoryError) Error() string { return e.message }
func (e *teamRepositoryError) Code() string  { return e.code }

var (
	ErrTeamDoesNotExist  = &teamRepositoryError{code: "not_found", message: "team not found"}
	ErrTeamOwnerRequired = &teamRepositoryError{code: "permission_denied", message: "team owner required"}
	ErrTeamMemberExists  = &teamRepositoryError{code: "already_exists", message: "team member already exists"}
	ErrTeamUserNotFound  = &teamRepositoryError{code: "not_found", message: "user not found"}
	ErrLastTeamOwner     = &teamRepositoryError{code: "failed_precondition", message: "team must have an owner"}
)

func (r *TeamRepository) FindAll(ctx context.Context, actorID uuid.UUID) ([]model.Team, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, t.created_at, t.updated_at
		FROM teams AS t
		JOIN team_memberships AS tm ON tm.team_id = t.id
		WHERE tm.user_id = $1
		ORDER BY t.created_at DESC, t.id DESC
	`, actorID)
	if err != nil {
		return nil, fmt.Errorf("query user teams: %w", err)
	}
	defer rows.Close()

	teams := make([]model.Team, 0)
	for rows.Next() {
		team, err := scanTeam(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user team: %w", err)
		}
		teams = append(teams, *team)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user teams: %w", err)
	}
	return teams, nil
}

func (r *TeamRepository) Create(ctx context.Context, actorID uuid.UUID, name string) (*model.Team, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create team: %w", err)
	}
	defer tx.Rollback(ctx)

	team, err := scanTeam(tx.QueryRow(ctx, `
		INSERT INTO teams (name)
		VALUES ($1)
		RETURNING id, name, created_at, updated_at
	`, name))
	if err != nil {
		return nil, fmt.Errorf("insert team: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO team_memberships (team_id, user_id, role)
		VALUES ($1, $2, $3)
	`, team.ID, actorID, model.TeamRoleOwner)
	if err != nil {
		return nil, fmt.Errorf("create team owner membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create team: %w", err)
	}
	return team, nil
}

func (r *TeamRepository) FindMembers(ctx context.Context, teamID uuid.UUID) ([]model.TeamMemberShip, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT team_id, user_id, role, joined_at
		FROM team_memberships
		WHERE team_id = $1
		ORDER BY joined_at, user_id
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("query team members: %w", err)
	}
	defer rows.Close()

	members := make([]model.TeamMemberShip, 0)
	for rows.Next() {
		member, err := scanTeamMember(rows)
		if err != nil {
			return nil, fmt.Errorf("scan team member: %w", err)
		}
		members = append(members, *member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate team members: %w", err)
	}
	return members, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {
	tx, err := r.beginOwnerMutation(ctx, actorID, teamID)
	if err != nil {
		return nil, fmt.Errorf("prepare add team member: %w", err)
	}
	defer tx.Rollback(ctx)

	member, err := scanTeamMember(tx.QueryRow(ctx, `
		INSERT INTO team_memberships (team_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING team_id, user_id, role, joined_at
	`, teamID, userID, role))
	if err != nil {
		return nil, membershipInsertError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add team member: %w", err)
	}
	return member, nil
}

func (r *TeamRepository) UpdateMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error) {
	tx, err := r.beginOwnerMutation(ctx, actorID, teamID)
	if err != nil {
		return nil, fmt.Errorf("prepare update team member: %w", err)
	}
	defer tx.Rollback(ctx)

	currentRole, found, err := memberRoleForUpdate(ctx, tx, teamID, userID)
	if err != nil {
		return nil, fmt.Errorf("find team member: %w", err)
	}
	if !found {
		return nil, nil
	}
	if currentRole == model.TeamRoleOwner && role != model.TeamRoleOwner {
		if err := requireAnotherOwner(ctx, tx, teamID); err != nil {
			return nil, err
		}
	}

	member, err := scanTeamMember(tx.QueryRow(ctx, `
		UPDATE team_memberships
		SET role = $3
		WHERE team_id = $1 AND user_id = $2
		RETURNING team_id, user_id, role, joined_at
	`, teamID, userID, role))
	if err != nil {
		return nil, fmt.Errorf("update team member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update team member: %w", err)
	}
	return member, nil
}

func (r *TeamRepository) DeleteMember(ctx context.Context, actorID, teamID, userID uuid.UUID) (bool, error) {
	tx, err := r.beginOwnerMutation(ctx, actorID, teamID)
	if err != nil {
		return false, fmt.Errorf("prepare delete team member: %w", err)
	}
	defer tx.Rollback(ctx)

	currentRole, found, err := memberRoleForUpdate(ctx, tx, teamID, userID)
	if err != nil {
		return false, fmt.Errorf("find team member: %w", err)
	}
	if !found {
		return false, nil
	}
	if currentRole == model.TeamRoleOwner {
		if err := requireAnotherOwner(ctx, tx, teamID); err != nil {
			return false, err
		}
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM team_memberships
		WHERE team_id = $1 AND user_id = $2
	`, teamID, userID)
	if err != nil {
		return false, fmt.Errorf("delete team member: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit delete team member: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *TeamRepository) beginOwnerMutation(ctx context.Context, actorID, teamID uuid.UUID) (pgx.Tx, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin team mutation: %w", err)
	}

	var lockedID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM teams WHERE id = $1 FOR UPDATE`, teamID).Scan(&lockedID)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = tx.Rollback(ctx)
		return nil, ErrTeamDoesNotExist
	}
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("lock team: %w", err)
	}

	var owner bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM team_memberships
			WHERE team_id = $1 AND user_id = $2 AND role = $3
		)
	`, teamID, actorID, model.TeamRoleOwner).Scan(&owner)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("check team owner: %w", err)
	}
	if !owner {
		_ = tx.Rollback(ctx)
		return nil, ErrTeamOwnerRequired
	}
	return tx, nil
}

func memberRoleForUpdate(ctx context.Context, tx pgx.Tx, teamID, userID uuid.UUID) (model.TeamRole, bool, error) {
	var role model.TeamRole
	err := tx.QueryRow(ctx, `
		SELECT role FROM team_memberships
		WHERE team_id = $1 AND user_id = $2
		FOR UPDATE
	`, teamID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return role, true, nil
}

func requireAnotherOwner(ctx context.Context, tx pgx.Tx, teamID uuid.UUID) error {
	var owners int
	err := tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM team_memberships
		WHERE team_id = $1 AND role = $2
	`, teamID, model.TeamRoleOwner).Scan(&owners)
	if err != nil {
		return fmt.Errorf("count team owners: %w", err)
	}
	if owners <= 1 {
		return ErrLastTeamOwner
	}
	return nil
}

func membershipInsertError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrTeamMemberExists
		case "23503":
			return ErrTeamUserNotFound
		}
	}
	return fmt.Errorf("insert team member: %w", err)
}

type teamRowScanner interface {
	Scan(dest ...any) error
}

func scanTeam(row teamRowScanner) (*model.Team, error) {
	var team model.Team
	err := row.Scan(&team.ID, &team.Name, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func scanTeamMember(row teamRowScanner) (*model.TeamMemberShip, error) {
	var member model.TeamMemberShip
	err := row.Scan(&member.TeamID, &member.UserID, &member.Role, &member.JoinedAt)
	if err != nil {
		return nil, err
	}
	return &member, nil
}
