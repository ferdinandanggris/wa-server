package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wa-server/internal/auth"
	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/service"
)

type UserService interface {
	Login(ctx context.Context, email, password string) (*service.LoginResponse, error)
	Create(ctx context.Context, input service.CreateUserInput) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	ListByCompany(ctx context.Context, companyID string) ([]models.User, error)
	Update(ctx context.Context, input service.UpdateUserInput) (*models.User, error)
	Delete(ctx context.Context, id string) error
}

type UserHandler struct {
	svc UserService
}

func NewUserHandler(svc UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	if body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "email and password are required"},
		})
		return
	}

	resp, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_CREDENTIALS", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": resp,
	})
}

func (h *UserHandler) createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "UNAUTHORIZED", "message": "not authenticated"},
		})
		return
	}

	var body struct {
		CompanyID string `json:"company_id"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		Name      string `json:"name"`
		Role      string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	if body.Email == "" || body.Password == "" || body.Name == "" || body.Role == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "email, password, name, and role are required"},
		})
		return
	}

	if claims.Role == string(models.RoleAdmin) && body.Role == string(models.RoleSuperadmin) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "FORBIDDEN", "message": "admin cannot create superadmin"},
		})
		return
	}

	companyID := body.CompanyID
	if claims.Role == string(models.RoleAdmin) {
		companyID = claims.CompanyID
	}
	if companyID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "company_id is required"},
		})
		return
	}

	user, err := h.svc.Create(r.Context(), service.CreateUserInput{
		CompanyID: companyID,
		Email:     body.Email,
		Password:  body.Password,
		Name:      body.Name,
		Role:      models.UserRole(body.Role),
	})
	if err != nil {
		slog.Error("failed to create user", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "data": user,
	})
}

func (h *UserHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "UNAUTHORIZED", "message": "not authenticated"},
		})
		return
	}

	companyID := claims.CompanyID
	if claims.Role == string(models.RoleSuperadmin) {
		companyID = r.URL.Query().Get("company_id")
	}

	users, err := h.svc.ListByCompany(r.Context(), companyID)
	if err != nil {
		slog.Error("failed to list users", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": "failed to list users"},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": users,
	})
}

func (h *UserHandler) getUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/users/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "user id is required"},
		})
		return
	}

	user, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "NOT_FOUND", "message": "user not found"},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": user,
	})
}

func (h *UserHandler) updateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/users/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "user id is required"},
		})
		return
	}

	var body struct {
		Name     *string `json:"name"`
		Email    *string `json:"email"`
		Password *string `json:"password"`
		Role     *string `json:"role"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	user, err := h.svc.Update(r.Context(), service.UpdateUserInput{
		ID:       id,
		Name:     body.Name,
		Email:    body.Email,
		Password: body.Password,
		Role:     body.Role,
		IsActive: body.IsActive,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": user,
	})
}

func (h *UserHandler) deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/users/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "user id is required"},
		})
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": map[string]string{"message": "user deleted"},
	})
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.HandleFunc("/api/v1/auth/login", h.login)
	mux.Handle("/api/v1/users", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.listUsers(w, r)
		case http.MethodPost:
			h.createUser(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
	mux.Handle("/api/v1/users/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.getUser(w, r)
		case http.MethodPut, http.MethodPatch:
			h.updateUser(w, r)
		case http.MethodDelete:
			h.deleteUser(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
}

func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	id = strings.TrimSuffix(id, "/")
	return id
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{
		"ok": false, "error": map[string]string{"code": "METHOD_NOT_ALLOWED", "message": "method not allowed"},
	})
}
