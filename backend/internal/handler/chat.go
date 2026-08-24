package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/juanbedoya/hnl-bank/backend/internal/middleware"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

// ChatHandler exposes the AI chat endpoint.
type ChatHandler struct {
	chat service.ChatService
}

// NewChatHandler builds a ChatHandler.
func NewChatHandler(chat service.ChatService) *ChatHandler {
	return &ChatHandler{chat: chat}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req service.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	res, err := h.chat.Chat(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, service.ErrChatNotConfigured) {
			response.Error(w, http.StatusServiceUnavailable, "AI chat is not configured", "chat_not_configured")
			return
		}
		response.Error(w, http.StatusInternalServerError, "chat error", "internal")
		return
	}
	response.JSON(w, http.StatusOK, res)
}

// Action executes a confirmed critical action (deposit/withdraw/transfer).
func (h *ChatHandler) Action(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var action map[string]any
	if err := json.NewDecoder(r.Body).Decode(&action); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	res, err := h.chat.Chat(r.Context(), userID, &service.ChatRequest{PendingAction: action})
	if err != nil {
		if errors.Is(err, service.ErrChatNotConfigured) {
			response.Error(w, http.StatusServiceUnavailable, "AI chat is not configured", "chat_not_configured")
			return
		}
		response.Error(w, http.StatusInternalServerError, "chat error", "internal")
		return
	}
	response.JSON(w, http.StatusOK, res)
}