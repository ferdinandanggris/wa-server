package handlers

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	mediaDir    = "uploads/media"
	maxFileSize = 16 << 20 // 16 MB
)

type MediaHandler struct {
	uploadDir string
}

func NewMediaHandler(uploadDir string) *MediaHandler {
	if uploadDir == "" {
		uploadDir = mediaDir
	}
	return &MediaHandler{uploadDir: uploadDir}
}

func (h *MediaHandler) upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "file too large or invalid multipart form"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "error": "file is required"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".bin"
	}

	dateDir := time.Now().Format("2006/01/02")
	uploadPath := filepath.Join(h.uploadDir, dateDir)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		slog.Error("failed to create upload directory", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to create upload directory"})
		return
	}

	fileID := fmt.Sprintf("%s-%s", time.Now().Format("150405"), generateUUIDShort())
	filename := fileID + ext
	destPath := filepath.Join(uploadPath, filename)

	dst, err := os.Create(destPath)
	if err != nil {
		slog.Error("failed to create file", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to save file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		slog.Error("failed to write file", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"ok": false, "error": "failed to write file"})
		return
	}

	fileType := detectContentType(ext)
	mediaPath := "/media/" + dateDir + "/" + filename

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"ok": true, "data": map[string]string{
			"media_id":  fileID,
			"file_path": mediaPath,
			"file_type": fileType,
		},
	})
}

func detectContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".pdf":
		return "application/pdf"
	case ".doc", ".docx":
		return "application/msword"
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	default:
		return "application/octet-stream"
	}
}

func generateUUIDShort() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func (h *MediaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/media/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.upload(w, r)
		} else {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]interface{}{"ok": false, "error": "method not allowed"})
		}
	})
}
