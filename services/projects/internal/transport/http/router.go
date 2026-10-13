package http

import "net/http"

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health/live", h.CheckLiveness)

	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("GET /api/v1/auth/me", h.Me)

	mux.HandleFunc("POST /api/v1/teams", h.CreateTeam)
	mux.HandleFunc("GET api/v1/teams", h.FindAllTeams)
	mux.HandleFunc("GET /v1/teams/{teamID}/members", h.FindTeamMembers)
	mux.HandleFunc("POST /v1/teams/{teamID}/members", h.AddTeamMember)
	mux.HandleFunc("PATCH /v1/teams/{teamID}/members/{userID}", h.UpdateTeamMember)
	mux.HandleFunc("DELETE /v1/teams/{teamID}/members/{userID}", h.DeleteTeamMember)

	mux.HandleFunc("POST /v1/teams/{teamID}/projects", h.CreateProject)
	mux.HandleFunc("GET /v1/teams/{teamID}/projects", h.FindAllProjects)
	mux.HandleFunc("GET /v1/projects/{projectID}", h.FindProjectById)
	mux.HandleFunc("PATCH /v1/projects/{projectID}", h.UpdateProject)
	mux.HandleFunc("POST /v1/projects/{projectID}/api-keys", h.CreateProjectAPIKey)
	mux.HandleFunc("GET /v1/projects/{projectID}/api-keys", h.GetProjectAPIKeys)
	mux.HandleFunc("DELETE /v1/projects/{projectID}/api-keys/{keyID}", h.RevokeAPIKey)

	return mux
}
