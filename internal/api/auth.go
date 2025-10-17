package api

import (
	"encoding/json"
	"net/http"
	"time"

	"photosync-backend/internal/auth"
	"photosync-backend/internal/models"
)

type AuthHandler struct {
	authenticator auth.Authenticator
	jwtManager    *auth.JWTManager
}

func NewAuthHandler(authenticator auth.Authenticator, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		authenticator: authenticator,
		jwtManager:    jwtManager,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	userInfo, err := h.authenticator.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := h.jwtManager.Generate(userInfo.Username, req.Password)
	if err != nil {
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	resp := models.LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
