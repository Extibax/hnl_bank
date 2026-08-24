package main

import (
	"bufio"
	"bytes"
	"context"
	"log"
	"os"

	"github.com/juanbedoya/hnl-bank/backend/internal/config"
	"github.com/juanbedoya/hnl-bank/backend/internal/mcp"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"github.com/juanbedoya/hnl-bank/backend/internal/service"
)

// main runs the HNL Bank banking tools as a Model Context Protocol (MCP)
// server over stdio (JSON-RPC 2.0). MCP-capable clients/agents can connect
// and call list_accounts, get_balance, get_transactions, make_deposit,
// make_withdrawal and make_transfer. The authenticated user id is passed as
// "user_id" inside each tool call arguments (per-session client).
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
	acctRepo := repository.NewAccountRepository(db, tb)
	txRepo := repository.NewTransactionRepository(db, tb)
	acctSvc := service.NewAccountService(acctRepo)
	txSvc := service.NewTransactionService(acctRepo, txRepo)
	srv := mcp.NewServer(acctSvc, txSvc)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		resp, _ := srv.HandleJSONRPC(line)
		if len(resp) > 0 {
			if _, err := os.Stdout.Write(append(resp, '\n')); err != nil {
				log.Printf("write: %v", err)
			}
		}
	}
}

// keep context imported for clarity
var _ = context.Background