package handler

import (
	"encoding/csv"
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
	txs, total, err := h.transactions.History(r.Context(), userID, accountID, limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	for i := range txs {
		txs[i].AmountStr = money.FromCents(txs[i].Amount)
	}
	response.JSON(w, http.StatusOK, map[string]any{"transactions": txs, "total": total})
}

// Export streams the user's transaction history as CSV.
func (h *TransactionHandler) Export(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	accountID := r.URL.Query().Get("account_id")

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"transacciones.csv\"")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"fecha", "tipo", "de", "para", "monto", "descripcion", "estado"})

	const pageSize = 100
	offset := 0
	for {
		txs, _, err := h.transactions.History(r.Context(), userID, accountID, pageSize, offset)
		if err != nil || len(txs) == 0 {
			break
		}
		for _, t := range txs {
			_ = cw.Write([]string{
				t.Timestamp.Format("2006-01-02 15:04:05"),
				t.Type,
				t.FromAccount,
				t.ToAccount,
				money.FromCents(t.Amount),
				t.Description,
				t.Status,
			})
		}
		if len(txs) < pageSize {
			break
		}
		offset += pageSize
	}
	cw.Flush()
}