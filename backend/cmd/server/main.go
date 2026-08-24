package main

import (
	"context"
	"log"
	"net/http"

	"github.com/juanbedoya/hnl-bank/backend/internal/config"
	"github.com/juanbedoya/hnl-bank/backend/internal/handler"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"github.com/juanbedoya/hnl-bank/backend/internal/seed"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := repository.NewPostgresDB(cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	client, err := repository.NewTigerBeetleClient(cfg.TigerBeetleAddr)
	if err != nil {
		log.Fatalf("tigerbeetle: %v", err)
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
		log.Fatalf("external account: %v", err)
	}
	if err := seed.Run(ctx, db, acctRepo, txRepo, tb); err != nil {
		log.Fatalf("seed: %v", err)
	}

	router := handler.BuildRouter(authSvc, acctSvc, txSvc, chatSvc)

	addr := ":" + cfg.Port
	log.Printf("HNL Bank backend listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}