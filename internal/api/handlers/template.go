package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/wa-server/internal/models"
)

type TemplateHandler struct {
	repo models.TemplateRepository
}

func NewTemplateHandler(repo models.TemplateRepository) *TemplateHandler {
	return &TemplateHandler{repo: repo}
}

type CreateTemplateRequest struct {
	WATemplateID  string `json:"wa_template_id"`
	Name          string `json:"name"`
	Language      string `json:"language"`
	Category      string `json:"category"`
	Content       string `json:"content"`
	HeaderType    string `json:"header_type,omitempty"`
	HeaderContent string `json:"header_content,omitempty"`
	IsVerified    bool   `json:"is_verified"`
	MetaStatus    string `json:"meta_status,omitempty"`
}

type UpdateTemplateRequest struct {
	WATemplateID  string `json:"wa_template_id"`
	Name          string `json:"name"`
	Language      string `json:"language"`
	Category      string `json:"category"`
	Content       string `json:"content"`
	HeaderType    string `json:"header_type,omitempty"`
	HeaderContent string `json:"header_content,omitempty"`
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
		WATemplateID:  req.WATemplateID,
		Name:          req.Name,
		Language:      req.Language,
		Category:      req.Category,
		Content:       req.Content,
		HeaderType:    req.HeaderType,
		HeaderContent: req.HeaderContent,
		IsVerified:    req.IsVerified,
		MetaStatus:    req.MetaStatus,
	}

	if err := h.repo.Create(r.Context(), tmpl); err != nil {
		slog.Error("failed to create template", "error", err)
		http.Error(w, "failed to create template", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tmpl)
}

func (h *TemplateHandler) list(w http.ResponseWriter, r *http.Request) {
	templates, err := h.repo.List(r.Context(), 100, 0)
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
	tmpl, err := h.repo.GetByID(r.Context(), id)
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
		ID:            id,
		WATemplateID:  req.WATemplateID,
		Name:          req.Name,
		Language:      req.Language,
		Category:      req.Category,
		Content:       req.Content,
		HeaderType:    req.HeaderType,
		HeaderContent: req.HeaderContent,
		IsVerified:    req.IsVerified,
		MetaStatus:    req.MetaStatus,
	}

	if err := h.repo.Update(r.Context(), tmpl); err != nil {
		slog.Error("failed to update template", "error", err)
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tmpl)
}

func (h *TemplateHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) extractID(path string) string {
	rest := strings.TrimPrefix(path, "/api/v1/templates/")
	if idx := strings.Index(rest, "/"); idx > 0 {
		rest = rest[:idx]
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
		id := h.extractID(r.URL.Path)
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
