package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/wa-server/internal/auth"
	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/service"
)

type CompanyService interface {
	Create(ctx context.Context, input service.CreateCompanyInput) (*models.Company, error)
	GetByID(ctx context.Context, id string) (*models.Company, error)
	List(ctx context.Context, limit, offset int) ([]models.Company, error)
	Update(ctx context.Context, input service.UpdateCompanyInput) (*models.Company, error)
	Delete(ctx context.Context, id string) error
}

type CompanyHandler struct {
	svc CompanyService
}

func NewCompanyHandler(svc CompanyService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

func (h *CompanyHandler) createCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	claims := auth.GetClaims(r.Context())
	if claims == nil || claims.Role != string(models.RoleSuperadmin) {
		writeJSON(w, http.StatusForbidden, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "FORBIDDEN", "message": "only superadmin can create companies"},
		})
		return
	}

	var body struct {
		Name        string `json:"name"`
		Code        string `json:"code"`
		PhoneNumber string `json:"phone_number"`
		Address     string `json:"address"`
		QuotaLimit  int    `json:"quota_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	company, err := h.svc.Create(r.Context(), service.CreateCompanyInput{
		Name:        body.Name,
		Code:        body.Code,
		PhoneNumber: body.PhoneNumber,
		Address:     body.Address,
		QuotaLimit:  body.QuotaLimit,
	})
	if err != nil {
		slog.Error("failed to create company", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "data": company,
	})
}

func (h *CompanyHandler) listCompanies(w http.ResponseWriter, r *http.Request) {
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

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if claims.Role == string(models.RoleAdmin) {
		company, err := h.svc.GetByID(r.Context(), claims.CompanyID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]interface{}{
				"ok": false, "error": map[string]string{"code": "NOT_FOUND", "message": "company not found"},
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok": true, "data": []models.Company{*company},
		})
		return
	}

	companies, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		slog.Error("failed to list companies", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": "failed to list companies"},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": companies,
	})
}

func (h *CompanyHandler) getCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/companies/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "company id is required"},
		})
		return
	}

	company, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "NOT_FOUND", "message": "company not found"},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": company,
	})
}

func (h *CompanyHandler) updateCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/companies/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "company id is required"},
		})
		return
	}

	var body struct {
		Name        *string `json:"name"`
		PhoneNumber *string `json:"phone_number"`
		Address     *string `json:"address"`
		IsActive    *bool   `json:"is_active"`
		QuotaLimit  *int    `json:"quota_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	company, err := h.svc.Update(r.Context(), service.UpdateCompanyInput{
		ID:          id,
		Name:        body.Name,
		PhoneNumber: body.PhoneNumber,
		Address:     body.Address,
		IsActive:    body.IsActive,
		QuotaLimit:  body.QuotaLimit,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": company,
	})
}

func (h *CompanyHandler) deleteCompany(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeMethodNotAllowed(w)
		return
	}

	id := extractID(r.URL.Path, "/api/v1/companies/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "company id is required"},
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
		"ok": true, "data": map[string]string{"message": "company deleted"},
	})
}

func (h *CompanyHandler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/companies", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.listCompanies(w, r)
		case http.MethodPost:
			h.createCompany(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
	mux.Handle("/api/v1/companies/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.getCompany(w, r)
		case http.MethodPut, http.MethodPatch:
			h.updateCompany(w, r)
		case http.MethodDelete:
			h.deleteCompany(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
}
