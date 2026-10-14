package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"LogFlux/services/projects/internal/model"
	"uuid"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (*model.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	Authenticate(ctx context.Context, token string) (uuid.UUID, error)
	FindUser(ctx context.Context, userID uuid.UUID) (*model.User, error)
}

type TeamService interface {
	FindAll(ctx context.Context, actorID uuid.UUID) ([]model.Team, error)
	Create(ctx context.Context, actorID uuid.UUID, name string) (*model.Team, error)
	FindMembers(ctx context.Context, actorID, teamID uuid.UUID) ([]model.TeamMemberShip, error)
	AddMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error)
	UpdateMember(ctx context.Context, actorID, teamID, userID uuid.UUID, role model.TeamRole) (*model.TeamMemberShip, error)
	DeleteMember(ctx context.Context, actorID, teamID, userID uuid.UUID) error
}

type ProjectService interface {
	FindAll(ctx context.Context, actorID, teamID uuid.UUID) ([]model.Project, error)
	Create(ctx context.Context, actorID, teamID uuid.UUID, name string) (*model.Project, error)
	FindByID(ctx context.Context, actorID, projectID uuid.UUID) (*model.Project, error)
	Update(ctx context.Context, actorID, projectID uuid.UUID, name *string, status *model.ProjectStatus) (*model.Project, error)
}

type APIKeyService interface {
	Create(ctx context.Context, actorID, projectID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (*model.APIKey, string, error)
	FindAll(ctx context.Context, actorID, projectID uuid.UUID) ([]model.APIKey, error)
	Revoke(ctx context.Context, actorID, projectID, keyID uuid.UUID) error
}

type Handler struct {
	auth     AuthService
	teams    TeamService
	projects ProjectService
	apiKeys  APIKeyService
}

func NewHandler(auth AuthService, teams TeamService, projects ProjectService, apiKeys APIKeyService) *Handler {
	return &Handler{
		auth:     auth,
		teams:    teams,
		projects: projects,
		apiKeys:  apiKeys,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	user, err := h.auth.Register(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if user == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, userResponse(user))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if token == "" {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, loginDTO{AccessToken: token, TokenType: "Bearer"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	user, err := h.auth.FindUser(r.Context(), actorID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if user == nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, userResponse(user))
}

func (h *Handler) FindAllTeams(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teams, err := h.teams.FindAll(r.Context(), actorID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]teamDTO, 0, len(teams))
	for i := range teams {
		items = append(items, teamResponse(&teams[i]))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var req createTeamRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	team, err := h.teams.Create(r.Context(), actorID, req.Name)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if team == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, teamResponse(team))
}

func (h *Handler) FindTeamMembers(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	members, err := h.teams.FindMembers(r.Context(), actorID, teamID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]memberDTO, 0, len(members))
	for i := range members {
		items = append(items, memberResponse(&members[i]))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	var req addTeamMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	userID, err := uuid.Parse(req.UserID)
	if err != nil || !validRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid user_id or role")
		return
	}
	member, err := h.teams.AddMember(r.Context(), actorID, teamID, userID, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if member == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, memberResponse(member))
}

func (h *Handler) UpdateTeamMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	userID, ok := pathID(w, r, "userID")
	if !ok {
		return
	}
	var req updateTeamMemberRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}
	member, err := h.teams.UpdateMember(r.Context(), actorID, teamID, userID, req.Role)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if member == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, memberResponse(member))
}

func (h *Handler) DeleteTeamMember(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	userID, ok := pathID(w, r, "userID")
	if !ok {
		return
	}
	if err := h.teams.DeleteMember(r.Context(), actorID, teamID, userID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) FindAllProjects(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	projects, err := h.projects.FindAll(r.Context(), actorID, teamID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]projectDTO, 0, len(projects))
	for i := range projects {
		items = append(items, projectResponse(&projects[i]))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) CreateProject(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	teamID, ok := pathID(w, r, "teamID")
	if !ok {
		return
	}
	var req createProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	project, err := h.projects.Create(r.Context(), actorID, teamID, req.Name)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if project == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, projectResponse(project))
}

func (h *Handler) FindProjectById(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	project, err := h.projects.FindByID(r.Context(), actorID, projectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, projectResponse(project))
}

func (h *Handler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	var req updateProjectRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == nil && req.Status == nil {
		writeError(w, http.StatusBadRequest, "name or status is required")
		return
	}
	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
		if *req.Name == "" {
			writeError(w, http.StatusBadRequest, "name cannot be empty")
			return
		}
	}
	if req.Status != nil && *req.Status != model.ProjectStatusActive && *req.Status != model.ProjectStatusArchived {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	project, err := h.projects.Update(r.Context(), actorID, projectID, req.Name, req.Status)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if project == nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, projectResponse(project))
}

func (h *Handler) CreateProjectAPIKey(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	var req createAPIKeyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Scopes) == 0 {
		writeError(w, http.StatusBadRequest, "name and scopes are required")
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		writeError(w, http.StatusBadRequest, "expires_at must be in the future")
		return
	}
	key, rawKey, err := h.apiKeys.Create(r.Context(), actorID, projectID, req.Name, req.Scopes, req.ExpiresAt)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	if key == nil || rawKey == "" {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response := apiKeyResponse(key)
	response.Key = rawKey
	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) GetProjectAPIKeys(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	keys, err := h.apiKeys.FindAll(r.Context(), actorID, projectID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}
	items := make([]apiKeyDTO, 0, len(keys))
	for i := range keys {
		items = append(items, apiKeyResponse(&keys[i]))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	actorID, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	projectID, ok := pathID(w, r, "projectID")
	if !ok {
		return
	}
	keyID, ok := pathID(w, r, "keyID")
	if !ok {
		return
	}
	if err := h.apiKeys.Revoke(r.Context(), actorID, projectID, keyID); err != nil {
		writeServiceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CheckLiveness(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) || strings.TrimSpace(header[len(prefix):]) == "" {
		writeError(w, http.StatusUnauthorized, "bearer token required")
		return uuid.UUID{}, false
	}
	userID, err := h.auth.Authenticate(r.Context(), header[len(prefix):])
	if err != nil {
		writeServiceError(w, r, err)
		return uuid.UUID{}, false
	}
	return userID, true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.UUID{}, false
	}
	return id, true
}

func validRole(role model.TeamRole) bool {
	return role == model.TeamRoleOwner || role == model.TeamRoleMember || role == model.TeamRoleViewer
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
		} else {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
		}
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON object")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorDTO{Error: message})
}

type codedError interface {
	Code() string
}

func writeServiceError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	message := "internal server error"

	var coded codedError
	if errors.As(err, &coded) {
		switch coded.Code() {
		case "invalid_argument":
			statusCode, message = http.StatusBadRequest, "invalid request"
		case "unauthenticated":
			statusCode, message = http.StatusUnauthorized, "unauthorized"
		case "permission_denied":
			statusCode, message = http.StatusForbidden, "access denied"
		case "not_found":
			statusCode, message = http.StatusNotFound, "not found"
		case "already_exists":
			statusCode, message = http.StatusConflict, "already exists"
		case "failed_precondition":
			statusCode, message = http.StatusConflict, "operation conflicts with current state"
		}
	}

	if statusCode == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "http_service_error",
			"method", r.Method,
			"path", r.URL.Path,
			"error", err,
		)
	}

	writeError(w, statusCode, message)
}
