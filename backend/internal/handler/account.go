package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/juanbedoya/hnl-bank/backend/internal/middleware"
	"github.com/juanbedoya/hnl-bank/backend/internal/money"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

// AccountHandler exposes account listing and balance endpoints.
type AccountHandler struct {
	accounts service.AccountService
}

// NewAccountHandler builds an AccountHandler.
func NewAccountHandler(accounts service.AccountService) *AccountHandler {
	return &AccountHandler{accounts: accounts}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	accs, err := h.accounts.GetAccounts(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "could not list accounts", "internal")
		return
	}
	out := make([]model.AccountResponse, 0, len(accs))
	for _, a := range accs {
		out = append(out, model.AccountResponse{
			ID:            a.ID,
			AccountNumber: a.AccountNumber,
			AccountType:   a.AccountType,
			Currency:      a.Currency,
			Balance:       money.FromCents(a.Balance),
			CreatedAt:     a.CreatedAt,
		})
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *AccountHandler) Balance(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	accountID := chi.URLParam(r, "id")
	balance, currency, err := h.accounts.GetBalance(r.Context(), userID, accountID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, model.AccountBalance{Balance: money.FromCents(balance), Currency: currency})
}

// writeServiceError maps service errors to HTTP responses.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		response.Error(w, http.StatusForbidden, "not authorized", "forbidden")
	case errors.Is(err, service.ErrAccountNotFound), errors.Is(err, service.ErrDestinationNotFound):
		response.Error(w, http.StatusNotFound, err.Error(), "not_found")
	case errors.Is(err, service.ErrInsufficientFunds):
		response.Error(w, http.StatusBadRequest, err.Error(), "insufficient_funds")
	case errors.Is(err, service.ErrNegativeAmount):
		response.Error(w, http.StatusBadRequest, err.Error(), "validation")
	case errors.Is(err, service.ErrSameAccount):
		response.Error(w, http.StatusBadRequest, err.Error(), "validation")
	default:
		response.Error(w, http.StatusInternalServerError, "internal server error", "internal")
	}
}
