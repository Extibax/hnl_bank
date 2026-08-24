package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

// AuthHandler exposes authentication endpoints.
type AuthHandler struct {
	auth service.AuthService
}

// NewAuthHandler builds an AuthHandler.
func NewAuthHandler(auth service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.FullName = strings.TrimSpace(req.FullName)
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		response.Error(w, http.StatusBadRequest, "email, password and full_name are required", "validation")
		return
	}
	u, err := h.auth.Register(r.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		if errors.Is(err, service.ErrEmailExists) {
			response.Error(w, http.StatusConflict, "email already registered", "email_exists")
			return
		}
		response.Error(w, http.StatusInternalServerError, "could not create user", "internal")
		return
	}
	response.JSON(w, http.StatusCreated, u)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	u, token, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "invalid credentials", "invalid_credentials")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	_ = h.auth.Logout(r.Context(), token)
	response.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
