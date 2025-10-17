package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"photosync-backend/internal/auth"
	"photosync-backend/internal/models"
	"photosync-backend/internal/storage"
)

type PhotoHandler struct {
	pool       *storage.GenericConnectionPool
	backend    storage.StorageBackend
	jwtManager *auth.JWTManager
}

func NewPhotoHandler(pool *storage.GenericConnectionPool, backend storage.StorageBackend, jwtManager *auth.JWTManager) *PhotoHandler {
	return &PhotoHandler{
		pool:       pool,
		backend:    backend,
		jwtManager: jwtManager,
	}
}

func (h *PhotoHandler) ListPhotos(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.JWTClaims)

	password, err := h.jwtManager.DecryptPassword(claims.EncryptedPassword)
	if err != nil {
		http.Error(w, "authentication error", http.StatusUnauthorized)
		return
	}

	conn, err := h.pool.GetConnection(claims.Username, password)
	if err != nil {
		http.Error(w, "failed to connect to storage", http.StatusInternalServerError)
		return
	}

	files, err := h.backend.List(conn, claims.Username)
	if err != nil {
		http.Error(w, "failed to list photos", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *PhotoHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.JWTClaims)

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "no photo provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	password, err := h.jwtManager.DecryptPassword(claims.EncryptedPassword)
	if err != nil {
		http.Error(w, "authentication error", http.StatusUnauthorized)
		return
	}

	conn, err := h.pool.GetConnection(claims.Username, password)
	if err != nil {
		http.Error(w, "failed to connect to storage", http.StatusInternalServerError)
		return
	}

	err = h.backend.Upload(conn, claims.Username, header.Filename, file)
	if err != nil {
		http.Error(w, "failed to upload photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.UploadResponse{
		Message:  "photo uploaded successfully",
		Filename: header.Filename,
	})
}

func (h *PhotoHandler) DownloadPhoto(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	photoID := chi.URLParam(r, "id")

	password, err := h.jwtManager.DecryptPassword(claims.EncryptedPassword)
	if err != nil {
		http.Error(w, "authentication error", http.StatusUnauthorized)
		return
	}

	conn, err := h.pool.GetConnection(claims.Username, password)
	if err != nil {
		http.Error(w, "failed to connect to storage", http.StatusInternalServerError)
		return
	}

	data, err := h.backend.Download(conn, claims.Username, photoID)
	if err != nil {
		http.Error(w, "photo not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", photoID))
	w.Write(data)
}

func (h *PhotoHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	photoID := chi.URLParam(r, "id")

	password, err := h.jwtManager.DecryptPassword(claims.EncryptedPassword)
	if err != nil {
		http.Error(w, "authentication error", http.StatusUnauthorized)
		return
	}

	conn, err := h.pool.GetConnection(claims.Username, password)
	if err != nil {
		http.Error(w, "failed to connect to storage", http.StatusInternalServerError)
		return
	}

	err = h.backend.Delete(conn, claims.Username, photoID)
	if err != nil {
		http.Error(w, "failed to delete photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PhotoHandler) GetPhotoInfo(w http.ResponseWriter, r *http.Request) {
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	photoID := chi.URLParam(r, "id")

	password, err := h.jwtManager.DecryptPassword(claims.EncryptedPassword)
	if err != nil {
		http.Error(w, "authentication error", http.StatusUnauthorized)
		return
	}

	conn, err := h.pool.GetConnection(claims.Username, password)
	if err != nil {
		http.Error(w, "failed to connect to storage", http.StatusInternalServerError)
		return
	}

	files, err := h.backend.List(conn, claims.Username)
	if err != nil {
		http.Error(w, "failed to get photo info", http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		if file.Name == photoID {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(file)
			return
		}
	}

	http.Error(w, "photo not found", http.StatusNotFound)
}

func (h *PhotoHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	h.DownloadPhoto(w, r)
}
