package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrChatNotConfigured indicates the AI chat is unavailable (missing API key).
var ErrChatNotConfigured = errors.New("AI chat no configurado: falta OPENROUTER_API_KEY")

// ChatMessage is a single exchange in the conversation history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the inbound chat payload.
type ChatRequest struct {
	Message       string         `json:"message"`
	History       []ChatMessage  `json:"history"`
	PendingAction map[string]any `json:"pending_action,omitempty"`
}

// ChatResult is the outbound chat response.
type ChatResult struct {
	Message              string         `json:"message"`
	RequiresConfirmation bool           `json:"requires_confirmation,omitempty"`
	PendingAction        map[string]any `json:"pending_action,omitempty"`
}

// ChatService drives the AI assistant via OpenRouter tool-use.
type ChatService interface {
	Chat(ctx context.Context, userID string, req *ChatRequest) (*ChatResult, error)
}

type chatService struct {
	apiKey       string
	model        string
	accounts     AccountService
	transactions TransactionService
	http         *http.Client
}

// NewChatService builds a ChatService. If apiKey is empty, the service returns
// ErrChatNotConfigured on every call.
func NewChatService(apiKey, model string, accounts AccountService, transactions TransactionService) ChatService {
	return &chatService{
		apiKey:       apiKey,
		model:        model,
		accounts:     accounts,
		transactions: transactions,
		http:         &http.Client{Timeout: 90 * time.Second},
	}
}

var allTools = []map[string]any{
	{"type":"function","function":map[string]any{"name":"list_accounts","description":"Lista las cuentas del usuario con saldos.","parameters":map[string]any{"type":"object","properties":map[string]any{},"required":[]string{}}}},
	{"type":"function","function":map[string]any{"name":"get_balance","description":"Devuelve el saldo de una cuenta.","parameters":map[string]any{"type":"object","properties":map[string]any{"account_id":map[string]any{"type":"string"}},"required":[]string{"account_id"}}}},
	{"type":"function","function":map[string]any{"name":"get_transactions","description":"Historial de transacciones de una cuenta.","parameters":map[string]any{"type":"object","properties":map[string]any{"account_id":map[string]any{"type":"string"},"limit":map[string]any{"type":"integer"},"offset":map[string]any{"type":"integer"}},"required":[]string{}}}},
	{"type":"function","function":map[string]any{"name":"make_deposit","description":"Deposita dinero en una cuenta. Accion critica.","parameters":map[string]any{"type":"object","properties":map[string]any{"account_id":map[string]any{"type":"string"},"amount":map[string]any{"type":"string","description":"monto en USD"}},"required":[]string{"account_id","amount"}}}},
	{"type":"function","function":map[string]any{"name":"make_withdrawal","description":"Retira dinero de una cuenta. Accion critica.","parameters":map[string]any{"type":"object","properties":map[string]any{"account_id":map[string]any{"type":"string"},"amount":map[string]any{"type":"string"}},"required":[]string{"account_id","amount"}}}},
	{"type":"function","function":map[string]any{"name":"make_transfer","description":"Transfiere dinero entre cuentas. Accion critica.","parameters":map[string]any{"type":"object","properties":map[string]any{"from_account":map[string]any{"type":"string"},"to_account":map[string]any{"type":"string"},"amount":map[string]any{"type":"string"}},"required":[]string{"from_account","to_account","amount"}}}},
}

var criticalTools = map[string]bool{"make_deposit": true, "make_withdrawal": true, "make_transfer": true}

func (s *chatService) Chat(ctx context.Context, userID string, req *ChatRequest) (*ChatResult, error) {
	if s.apiKey == "" {
		return nil, ErrChatNotConfigured
	}
	if req.PendingAction != nil {
		return s.executeAction(ctx, userID, req.PendingAction)
	}
	messages := s.buildMessages(req.Message, req.History)
	for i := 0; i < 5; i++ {
		msg, err := s.callOpenRouter(ctx, messages)
		if err != nil {
			return nil, err
		}
		toolCalls := msg.toToolCalls()
		if len(toolCalls) == 0 {
			return &ChatResult{Message: msg.Content}, nil
		}
		for _, tc := range toolCalls {
			if criticalTools[tc.Name] {
				return &ChatResult{
					Message:              "¿Confirmar la operación?",
					RequiresConfirmation: true,
					PendingAction:        map[string]any{"tool": tc.Name, "args": tc.Args},
				}, nil
			}
		}
		assistant := map[string]any{"role": "assistant", "content": msg.Content, "tool_calls": msg.RawToolCalls}
		messages = append(messages, assistant)
		for _, tc := range toolCalls {
			res := s.execute(ctx, userID, tc.Name, tc.Args)
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": tc.ID, "content": res})
		}
	}
	return &ChatResult{Message: "Lo siento, no pude completar la solicitud."}, nil
}

type toolCall struct {
	Name string
	Args map[string]any
	ID   string
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIMessage struct {
	Content       string           `json:"content"`
	ToolCalls     []openAIToolCall `json:"tool_calls,omitempty"`
	RawToolCalls  []map[string]any `json:"-"`
}

func (m *openAIMessage) toToolCalls() []toolCall {
	var out []toolCall
	for _, tc := range m.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		out = append(out, toolCall{Name: tc.Function.Name, Args: args, ID: tc.ID})
	}
	return out
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (s *chatService) callOpenRouter(ctx context.Context, messages []map[string]any) (*openAIMessage, error) {
	body, _ := json.Marshal(map[string]any{"model": s.model, "messages": messages, "tools": allTools})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := s.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("openrouter error: " + strings.TrimSpace(string(data)))
	}
	var parsed openAIResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Choices) == 0 {
		return &openAIMessage{}, nil
	}
	return &parsed.Choices[0].Message, nil
}

func (s *chatService) buildMessages(message string, history []ChatMessage) []map[string]any {
	sys := "Eres el asistente de HNL Bank, un banco digital. Ayudas al usuario autenticado a consultar saldo, ver transacciones y hacer depositos, retiros y transferencias. Antes de ejecutar una accion critica (deposito, retiro o transferencia) usa siempre la herramienta correspondiente. Responde de forma concisa en espanol."
	out := []map[string]any{{"role": "system", "content": sys}}
	for _, h := range history {
		if h.Role != "" && h.Content != "" {
			out = append(out, map[string]any{"role": h.Role, "content": h.Content})
		}
	}
	out = append(out, map[string]any{"role": "user", "content": message})
	return out
}

func (s *chatService) execute(ctx context.Context, userID, name string, args map[string]any) string {
	switch name {
	case "list_accounts":
		accs, err := s.accounts.GetAccounts(ctx, userID)
		if err != nil {
			return `{"error": "` + err.Error() + `"}`
		}
		type row struct {
			AccountNumber string `json:"account_number"`
			Type          string `json:"account_type"`
			Balance       string `json:"balance"`
		}
		var rows []row
		for _, a := range accs {
			rows = append(rows, row{AccountNumber: a.AccountNumber, Type: a.AccountType, Balance: moneyFromCents(a.Balance)})
		}
		b, _ := json.Marshal(rows)
		return string(b)
	case "get_balance":
		acc, _ := args["account_id"].(string)
		if acc == "" {
			return `{"error": "missing account_id"}`
		}
		balance, currency, err := s.accounts.GetBalance(ctx, userID, acc)
		if err != nil {
			return `{"error": "` + err.Error() + `"}`
		}
		return `{"balance": "` + moneyFromCents(balance) + `", "currency": "` + currency + `"}`
	case "get_transactions":
		acc, _ := args["account_id"].(string)
		limit := 20
		if v, ok := args["limit"].(float64); ok && v > 0 {
			limit = int(v)
		}
		txs, _, err := s.transactions.History(ctx, userID, acc, limit, 0)
		if err != nil {
			return `{"error": "` + err.Error() + `"}`
		}
		b, _ := json.Marshal(txs)
		return string(b)
	default:
		return `{"error": "unknown tool"}`
	}
}

func (s *chatService) executeAction(ctx context.Context, userID string, action map[string]any) (*ChatResult, error) {
	tool, _ := action["tool"].(string)
	args, _ := action["args"].(map[string]any)
	if tool == "" {
		return &ChatResult{Message: "Acción no reconocida."}, nil
	}
	if args == nil {
		args = map[string]any{}
	}
	amountStr, _ := args["amount"].(string)
	amount, err := moneyToCents(amountStr)
	if err != nil {
		return &ChatResult{Message: "Monto inválido."}, nil
	}
	switch tool {
	case "make_deposit":
		acct, _ := args["account_id"].(string)
		if err := s.transactions.Deposit(ctx, userID, acct, amount); err != nil {
			return &ChatResult{Message: "Error: " + err.Error()}, nil
		}
		return &ChatResult{Message: "Depósito realizado correctamente."}, nil
	case "make_withdrawal":
		acct, _ := args["account_id"].(string)
		if err := s.transactions.Withdraw(ctx, userID, acct, amount); err != nil {
			return &ChatResult{Message: "Error: " + err.Error()}, nil
		}
		return &ChatResult{Message: "Retiro realizado correctamente."}, nil
	case "make_transfer":
		from, _ := args["from_account"].(string)
		to, _ := args["to_account"].(string)
		if err := s.transactions.Transfer(ctx, userID, from, to, amount); err != nil {
			return &ChatResult{Message: "Error: " + err.Error()}, nil
		}
		return &ChatResult{Message: "Transferencia realizada correctamente."}, nil
	}
	return &ChatResult{Message: "Acción no reconocida."}, nil
}