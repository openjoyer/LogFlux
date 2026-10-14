package model

import (
	"time"
	"uuid"
)

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "OWNER"
	TeamRoleMember TeamRole = "MEMBER"
	TeamRoleViewer TeamRole = "VIEWER"
)

type Team struct {
	ID        uuid.UUID
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	UserID    uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
}

type TeamMemberShip struct {
	TeamID   uuid.UUID
	UserID   uuid.UUID
	Role     TeamRole
	JoinedAt time.Time
}

type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
)

type Project struct {
	ID        uuid.UUID
	Name      string
	TeamID    uuid.UUID
	Status    ProjectStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type APIKey struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	Name      string
	KeyHash   []byte
	Scopes    []string
	CreatedAt time.Time
	ExpiresAt *time.Time
	RevokedAt *time.Time
}
