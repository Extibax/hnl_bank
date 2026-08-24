package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/juanbedoya/hnl-bank/backend/internal/config"
	"github.com/juanbedoya/hnl-bank/backend/internal/handler"
	"github.com/juanbedoya/hnl-bank/backend/internal/middleware"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"github.com/juanbedoya/hnl-bank/backend/internal/seed"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
)

func main() {
	cfg := config.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	fatalf := func(msg string, err error) {
		logger.Error(msg, "error", err)
		os.Exit(1)
	}

	db, err := repository.NewPostgresDB(cfg.PostgresDSN)
	if err != nil {
		fatalf("postgres", err)
	}
	defer db.Close()

	client, err := repository.NewTigerBeetleClient(cfg.TigerBeetleAddr)
	if err != nil {
		fatalf("tigerbeetle", err)
	}
	defer client.Close()

	tb := repository.NewTigerBeetleRepository(client)

	userRepo := repository.NewUserRepository(db)
	acctRepo := repository.NewAccountRepository(db, tb)
	txRepo := repository.NewTransactionRepository(db, tb)

	authSvc := service.NewAuthService(userRepo, acctRepo, cfg.JWTSecret)
	acctSvc := service.NewAccountService(acctRepo)
	txSvc := service.NewTransactionService(acctRepo, txRepo)
	chatSvc := service.NewChatService(cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL, acctSvc, txSvc)

	ctx := context.Background()
	if err := tb.EnsureExternalAccount(ctx); err != nil {
		fatalf("external account", err)
	}
	if err := seed.Run(ctx, db, acctRepo, txRepo, tb); err != nil {
		fatalf("seed", err)
	}

	rateLimit := middleware.RateLimit(cfg.RateLimitRequests, time.Duration(cfg.RateLimitWindowSecs)*time.Second)

	router := handler.BuildRouter(logger, rateLimit, authSvc, acctSvc, txSvc, chatSvc)

	addr := ":" + cfg.Port
	logger.Info("backend listening", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		fatalf("listen", err)
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}