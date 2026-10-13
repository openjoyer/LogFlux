package http

import (
	"time"

	"LogFlux/services/projects/internal/model"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type addTeamMemberRequest struct {
	UserID string         `json:"user_id"`
	Role   model.TeamRole `json:"role"`
}

type updateTeamMemberRequest struct {
	Role model.TeamRole `json:"role"`
}

type createProjectRequest struct {
	Name string `json:"name"`
}

type updateProjectRequest struct {
	Name   *string              `json:"name"`
	Status *model.ProjectStatus `json:"status"`
}

type createAPIKeyRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type loginDTO struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type errorDTO struct {
	Error string `json:"error"`
}

type userDTO struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type teamDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type memberDTO struct {
	TeamID   string         `json:"team_id"`
	UserID   string         `json:"user_id"`
	Role     model.TeamRole `json:"role"`
	JoinedAt time.Time      `json:"joined_at"`
}

type projectDTO struct {
	ID        string              `json:"id"`
	TeamID    string              `json:"team_id"`
	Name      string              `json:"name"`
	Status    model.ProjectStatus `json:"status"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
}

type apiKeyDTO struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at"`
	Key       string     `json:"key,omitempty"`
}

func userResponse(user *model.User) userDTO {
	return userDTO{
		ID: user.ID.String(), Email: user.Email,
		CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

func teamResponse(team *model.Team) teamDTO {
	return teamDTO{
		ID: team.ID.String(), Name: team.Name,
		CreatedAt: team.CreatedAt, UpdatedAt: team.UpdatedAt,
	}
}

func memberResponse(member *model.TeamMemberShip) memberDTO {
	return memberDTO{
		TeamID: member.TeamID.String(), UserID: member.UserID.String(),
		Role: member.Role, JoinedAt: member.JoinedAt,
	}
}

func projectResponse(project *model.Project) projectDTO {
	return projectDTO{
		ID: project.ID.String(), TeamID: project.TeamID.String(),
		Name: project.Name, Status: project.Status,
		CreatedAt: project.CreatedAt, UpdatedAt: project.UpdatedAt,
	}
}

func apiKeyResponse(key *model.APIKey) apiKeyDTO {
	return apiKeyDTO{
		ID: key.ID.String(), ProjectID: key.ProjectID.String(),
		Name: key.Name, Scopes: key.Scopes,
		CreatedAt: key.CreatedAt, ExpiresAt: key.ExpiresAt, RevokedAt: key.RevokedAt,
	}
}
