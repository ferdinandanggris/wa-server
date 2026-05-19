package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/wa-server/internal/auth"
	"github.com/wa-server/internal/models"
)

type PhoneNumberService interface {
	SyncFromMeta(ctx context.Context) (int, error)
	List(ctx context.Context) ([]models.PhoneNumber, error)
	AssignToCompany(ctx context.Context, id, companyID string) (*models.PhoneNumber, error)
	GetProfile(ctx context.Context, id string) (*models.WhatsAppBusinessProfile, error)
	UpdateProfile(ctx context.Context, id string, profile *models.WhatsAppBusinessProfile) error
}

type PhoneNumberHandler struct {
	svc PhoneNumberService
}

func NewPhoneNumberHandler(svc PhoneNumberService) *PhoneNumberHandler {
	return &PhoneNumberHandler{svc: svc}
}

func (h *PhoneNumberHandler) list(w http.ResponseWriter, r *http.Request) {
	numbers, err := h.svc.List(r.Context())
	if err != nil {
		slog.Error("failed to list phone numbers", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": "failed to list phone numbers"},
		})
		return
	}

	if numbers == nil {
		numbers = []models.PhoneNumber{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": numbers,
	})
}

func (h *PhoneNumberHandler) sync(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	synced, err := h.svc.SyncFromMeta(ctx)
	if err != nil {
		slog.Error("failed to sync phone numbers", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "SYNC_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": map[string]interface{}{"message": "phone numbers sync completed", "synced": synced},
	})
}

func (h *PhoneNumberHandler) assignToCompany(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/api/v1/phone-numbers/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "phone number id is required"},
		})
		return
	}

	id = strings.TrimSuffix(id, "/assign")

	var body struct {
		CompanyID string `json:"company_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}
	if body.CompanyID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "company_id is required"},
		})
		return
	}

	pn, err := h.svc.AssignToCompany(r.Context(), id, body.CompanyID)
	if err != nil {
		slog.Error("failed to assign phone number", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INTERNAL_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": pn,
	})
}

func (h *PhoneNumberHandler) getProfile(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/api/v1/phone-numbers/")
	id = strings.TrimSuffix(id, "/profile")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "phone number id is required"},
		})
		return
	}

	profile, err := h.svc.GetProfile(r.Context(), id)
	if err != nil {
		slog.Error("failed to get profile", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "PROFILE_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": profile,
	})
}

func (h *PhoneNumberHandler) updateProfile(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/api/v1/phone-numbers/")
	id = strings.TrimSuffix(id, "/profile")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "VALIDATION_ERROR", "message": "phone number id is required"},
		})
		return
	}

	var profile models.WhatsAppBusinessProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "INVALID_JSON", "message": "invalid request body"},
		})
		return
	}

	if err := h.svc.UpdateProfile(r.Context(), id, &profile); err != nil {
		slog.Error("failed to update profile", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok": false, "error": map[string]string{"code": "PROFILE_ERROR", "message": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true, "data": map[string]string{"message": "profile updated successfully"},
	})
}

func (h *PhoneNumberHandler) RegisterRoutes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.Handle("/api/v1/phone-numbers", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.list(w, r)
		case http.MethodPost:
			h.sync(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
	mux.Handle("/api/v1/phone-numbers/", authMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := auth.GetClaims(r.Context())
		if claims == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"ok": false, "error": map[string]string{"code": "UNAUTHORIZED", "message": "authentication required"},
			})
			return
		}

		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/assign") && r.Method == http.MethodPost:
			if claims.Role != string(models.RoleSuperadmin) {
				writeJSON(w, http.StatusForbidden, map[string]interface{}{
					"ok": false, "error": map[string]string{"code": "FORBIDDEN", "message": "only superadmin can assign phone numbers"},
				})
				return
			}
			h.assignToCompany(w, r)
		case strings.HasSuffix(path, "/profile") && r.Method == http.MethodGet:
			h.getProfile(w, r)
		case strings.HasSuffix(path, "/profile") && (r.Method == http.MethodPut || r.Method == http.MethodPatch):
			h.updateProfile(w, r)
		default:
			writeMethodNotAllowed(w)
		}
	})))
}
