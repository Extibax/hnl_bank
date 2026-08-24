package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/juanbedoya/hnl-bank/backend/internal/middleware"
	"github.com/juanbedoya/hnl-bank/backend/internal/money"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

// TransactionHandler exposes deposit, withdraw, transfer and history endpoints.
type TransactionHandler struct {
	transactions service.TransactionService
}

// NewTransactionHandler builds a TransactionHandler.
func NewTransactionHandler(transactions service.TransactionService) *TransactionHandler {
	return &TransactionHandler{transactions: transactions}
}

type amountRequest struct {
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
}

type transferRequest struct {
	FromAccount string `json:"from_account"`
	ToAccount   string `json:"to_account"`
	Amount      string `json:"amount"`
}

func (h *TransactionHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	cents, err := money.ToCents(req.Amount)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid amount", "validation")
		return
	}
	if err := h.transactions.Deposit(r.Context(), userID, req.AccountID, cents); err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"status": model.StatusCompleted, "type": model.TypeDeposit})
}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req amountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	cents, err := money.ToCents(req.Amount)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid amount", "validation")
		return
	}
	if err := h.transactions.Withdraw(r.Context(), userID, req.AccountID, cents); err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"status": model.StatusCompleted, "type": model.TypeWithdraw})
}

func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	var req transferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}
	cents, err := money.ToCents(req.Amount)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid amount", "validation")
		return
	}
	if err := h.transactions.Transfer(r.Context(), userID, req.FromAccount, req.ToAccount, cents); err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"status": model.StatusCompleted, "type": model.TypeTransfer})
}

func (h *TransactionHandler) History(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	accountID := chi.URLParam(r, "account_id")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	txs, _, err := h.transactions.History(r.Context(), userID, accountID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	for i := range txs {
		txs[i].AmountStr = money.FromCents(txs[i].Amount)
	}
	response.JSON(w, http.StatusOK, map[string]any{"transactions": txs})
}