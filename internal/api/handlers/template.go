package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wa-server/internal/models"
	"github.com/wa-server/internal/service"
)

type TemplateHandler struct {
	svc *service.TemplateService
}

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

type CreateTemplateRequest struct {
	Name          string `json:"name"`
	Language      string `json:"language"`
	Category      string `json:"category"`
	Content       string `json:"content"`
	HeaderType    string `json:"header_type,omitempty"`
	HeaderContent string `json:"header_content,omitempty"`
	FooterText    string `json:"footer_text,omitempty"`
	Buttons       string `json:"buttons,omitempty"`
	BodyComponents string `json:"body_components,omitempty"`
}

type UpdateTemplateRequest struct {
	Name          string `json:"name"`
	Language      string `json:"language"`
	Category      string `json:"category"`
	Content       string `json:"content"`
	HeaderType    string `json:"header_type,omitempty"`
	HeaderContent string `json:"header_content,omitempty"`
	FooterText    string `json:"footer_text,omitempty"`
	Buttons       string `json:"buttons,omitempty"`
	BodyComponents string `json:"body_components,omitempty"`
	IsVerified    bool   `json:"is_verified"`
	MetaStatus    string `json:"meta_status,omitempty"`
}

func (h *TemplateHandler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.Name == "" || req.Content == "" {
		http.Error(w, "name and content are required", http.StatusBadRequest)
		return
	}

	tmpl := &models.Template{
		Name:           req.Name,
		Language:       req.Language,
		Category:       req.Category,
		Content:        req.Content,
		HeaderType:     req.HeaderType,
		HeaderContent:  req.HeaderContent,
		FooterText:     req.FooterText,
		Buttons:        req.Buttons,
		BodyComponents: req.BodyComponents,
	}

	if err := h.svc.CreateAndSync(r.Context(), tmpl); err != nil {
		slog.Error("failed to create template", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tmpl)
}

func (h *TemplateHandler) list(w http.ResponseWriter, r *http.Request) {
	templates, err := h.svc.List(r.Context(), 100, 0)
	if err != nil {
		slog.Error("failed to list templates", "error", err)
		http.Error(w, "failed to list templates", http.StatusInternalServerError)
		return
	}

	if templates == nil {
		templates = []models.Template{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

func (h *TemplateHandler) getByID(w http.ResponseWriter, r *http.Request, id string) {
	tmpl, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tmpl)
}

func (h *TemplateHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	tmpl := &models.Template{
		ID:             id,
		Name:           req.Name,
		Language:       req.Language,
		Category:       req.Category,
		Content:        req.Content,
		HeaderType:     req.HeaderType,
		HeaderContent:  req.HeaderContent,
		FooterText:     req.FooterText,
		Buttons:        req.Buttons,
		BodyComponents: req.BodyComponents,
		IsVerified:     req.IsVerified,
		MetaStatus:     req.MetaStatus,
	}

	if err := h.svc.Update(r.Context(), tmpl); err != nil {
		slog.Error("failed to update template", "error", err)
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tmpl)
}

func (h *TemplateHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) syncAll(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.SyncAll(r.Context())
	if err != nil {
		slog.Error("failed to sync templates", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "sync completed",
		"synced":  count,
	})
}

func (h *TemplateHandler) syncStatus(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.SyncPendingStatus(r.Context())
	if err != nil {
		slog.Error("failed to sync pending status", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "status sync completed",
		"updated": count,
	})
}

func (h *TemplateHandler) extractID(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/templates")
	rest = strings.TrimPrefix(rest, "/")
	if idx := strings.Index(rest, "/"); idx > 0 {
		return rest[:idx]
	}
	return rest
}

func (h *TemplateHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.create(w, r)
		case http.MethodGet:
			h.list(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/v1/templates/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")

		if path == "sync" && r.Method == http.MethodPost {
			h.syncAll(w, r)
			return
		}
		if path == "sync/status" && r.Method == http.MethodPost {
			h.syncStatus(w, r)
			return
		}

		id := path
		if id == "" {
			http.Error(w, "template id is required", http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getByID(w, r, id)
		case http.MethodPut:
			h.update(w, r, id)
		case http.MethodDelete:
			h.delete(w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}
