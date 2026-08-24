package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/juanbedoya/hnl-bank/backend/internal/money"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
)

// AccountReader is the subset of the account service needed to expose tools.
type AccountReader interface {
	GetAccounts(ctx context.Context, userID string) ([]model.AccountWithBalance, error)
	GetBalance(ctx context.Context, userID, accountID string) (int64, string, error)
}

// TransactionExecutor is the subset of the transaction service needed to expose tools.
type TransactionExecutor interface {
	Deposit(ctx context.Context, userID, accountID string, amount int64) error
	Withdraw(ctx context.Context, userID, accountID string, amount int64) error
	Transfer(ctx context.Context, userID, fromAccountID, toAccount string, amount int64) error
	History(ctx context.Context, userID, accountID string, limit, offset int) ([]model.Transaction, int, error)
}

// Tool is an MCP tool definition exposed by the banking server.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// Server exposes the banking tools over the Model Context Protocol (MCP).
type Server struct {
	accounts     AccountReader
	transactions TransactionExecutor
}

// NewServer builds an MCP server wired to the banking services.
func NewServer(accounts AccountReader, transactions TransactionExecutor) *Server {
	return &Server{accounts: accounts, transactions: transactions}
}

// Tools returns the MCP tool definitions.
func (s *Server) Tools() []Tool {
	return []Tool{
		{Name: "list_accounts", Description: "Lista las cuentas del usuario con sus saldos.", InputSchema: schemaObj()},
		{Name: "get_balance", Description: "Devuelve el saldo de una cuenta en USD.", InputSchema: schemaObj(map[string]any{"account_id": schemaStr("id de la cuenta")})},
		{Name: "get_transactions", Description: "Historial de transacciones de una cuenta.", InputSchema: schemaObj(map[string]any{"account_id": schemaStr(""), "limit": schemaInt(), "offset": schemaInt()})},
		{Name: "make_deposit", Description: "Deposita dinero en una cuenta. Accion critica.", InputSchema: schemaObj(map[string]any{"account_id": schemaStr(""), "amount": schemaStr("monto en USD")})},
		{Name: "make_withdrawal", Description: "Retira dinero de una cuenta. Accion critica.", InputSchema: schemaObj(map[string]any{"account_id": schemaStr(""), "amount": schemaStr("monto en USD")})},
		{Name: "make_transfer", Description: "Transfiere dinero entre cuentas. Accion critica.", InputSchema: schemaObj(map[string]any{"from_account": schemaStr(""), "to_account": schemaStr(""), "amount": schemaStr("monto en USD")})},
	}
}

// CallTool executes an MCP tool for a given authenticated user.
func (s *Server) CallTool(ctx context.Context, userID, name string, args map[string]any) (string, error) {
	switch name {
	case "list_accounts":
		accs, err := s.accounts.GetAccounts(ctx, userID)
		if err != nil {
			return "", err
		}
		type row struct {
			AccountNumber string `json:"account_number"`
			Type          string `json:"account_type"`
			Balance       string `json:"balance"`
		}
		rows := make([]row, 0, len(accs))
		for _, a := range accs {
			rows = append(rows, row{AccountNumber: a.AccountNumber, Type: a.AccountType, Balance: money.FromCents(a.Balance)})
		}
		b, _ := json.Marshal(rows)
		return string(b), nil

	case "get_balance":
		acc, _ := args["account_id"].(string)
		if acc == "" {
			return "", fmt.Errorf("missing account_id")
		}
		balance, currency, err := s.accounts.GetBalance(ctx, userID, acc)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(map[string]string{"balance": money.FromCents(balance), "currency": currency})
		return string(b), nil

	case "get_transactions":
		acc, _ := args["account_id"].(string)
		limit := 20
		if v, ok := args["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		txs, _, err := s.transactions.History(ctx, userID, acc, limit, 0)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(txs)
		return string(b), nil

	case "make_deposit":
		return s.doDeposit(ctx, userID, args)
	case "make_withdrawal":
		return s.doWithdraw(ctx, userID, args)
	case "make_transfer":
		return s.doTransfer(ctx, userID, args)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *Server) doDeposit(ctx context.Context, userID string, args map[string]any) (string, error) {
	acc, _ := args["account_id"].(string)
	cents, err := money.ToCents(argStr(args["amount"]))
	if err != nil {
		return "", err
	}
	if err := s.transactions.Deposit(ctx, userID, acc, cents); err != nil {
		return "", err
	}
	return `{"ok":true,"type":"deposit"}`, nil
}

func (s *Server) doWithdraw(ctx context.Context, userID string, args map[string]any) (string, error) {
	acc, _ := args["account_id"].(string)
	cents, err := money.ToCents(argStr(args["amount"]))
	if err != nil {
		return "", err
	}
	if err := s.transactions.Withdraw(ctx, userID, acc, cents); err != nil {
		return "", err
	}
	return `{"ok":true,"type":"withdraw"}`, nil
}

func (s *Server) doTransfer(ctx context.Context, userID string, args map[string]any) (string, error) {
	from, _ := args["from_account"].(string)
	to, _ := args["to_account"].(string)
	cents, err := money.ToCents(argStr(args["amount"]))
	if err != nil {
		return "", err
	}
	if err := s.transactions.Transfer(ctx, userID, from, to, cents); err != nil {
		return "", err
	}
	return `{"ok":true,"type":"transfer"}`, nil
}

func argStr(a any) string {
	if s, ok := a.(string); ok {
		return s
	}
	return ""
}

func schemaObj(props ...map[string]any) map[string]any {
	p := map[string]any{}
	for _, mp := range props {
		for k, v := range mp {
			p[k] = v
		}
	}
	return map[string]any{"type": "object", "properties": p}
}

func schemaStr(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func schemaInt() map[string]any {
	return map[string]any{"type": "integer", "description": ""}
}