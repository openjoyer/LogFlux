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

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

type userRepositoryError struct {
	code    string
	message string
}

func (e *userRepositoryError) Error() string { return e.message }
func (e *userRepositoryError) Code() string  { return e.code }

var ErrEmailAlreadyRegistered = &userRepositoryError{
	code:    "already_exists",
	message: "email already registered",
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash string) (*model.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id, email, password_hash, created_at, updated_at
	`, email, passwordHash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyRegistered
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := scanUser(r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, created_at, updated_at
		FROM users
		WHERE id = $1
	`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by ID: %w", err)
	}
	return user, nil
}

type userRowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row userRowScanner) (*model.User, error) {
	var user model.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
