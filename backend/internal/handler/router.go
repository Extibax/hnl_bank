package handler

import (
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/chi/v5"

	"github.com/juanbedoya/hnl-bank/backend/internal/middleware"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
	"github.com/juanbedoya/hnl-bank/backend/pkg/response"
)

// BuildRouter wires all HTTP routes with Chi.
func BuildRouter(logger *slog.Logger, rateLimit func(http.Handler) http.Handler, auth service.AuthService, accounts service.AccountService, transactions service.TransactionService, chat service.ChatService) http.Handler {
	authH := NewAuthHandler(auth)
	accountH := NewAccountHandler(accounts)
	txH := NewTransactionHandler(transactions)
	chatH := NewChatHandler(chat)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.StructuredLogger(logger))
	r.Use(chimw.Recoverer)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	r.Route("/api/auth", func(sr chi.Router) {
		sr.Use(rateLimit)
		sr.Post("/register", authH.Register)
		sr.Post("/login", authH.Login)
		sr.Group(func(g chi.Router) {
			g.Use(middleware.JWTAuth(auth.ValidateToken))
			g.Post("/logout", authH.Logout)
		})
	})

	r.Group(func(g chi.Router) {
		g.Use(middleware.JWTAuth(auth.ValidateToken))
		g.Use(rateLimit)

		g.Get("/api/accounts", accountH.List)
		g.Get("/api/accounts/{id}/balance", accountH.Balance)

		g.Get("/api/transactions/export", txH.Export)
		g.Post("/api/transactions/deposit", txH.Deposit)
		g.Post("/api/transactions/withdraw", txH.Withdraw)
		g.Post("/api/transactions/transfer", txH.Transfer)
		g.Get("/api/transactions/{account_id}", txH.History)

		g.Post("/api/chat", chatH.Chat)
		g.Post("/api/chat/action", chatH.Action)
	})

	return r
}
