package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/juanbedoya/hnl-bank/backend/internal/mcp"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrChatNotConfigured indicates the AI chat is unavailable (missing API key).
var ErrChatNotConfigured = errors.New("AI chat no configurado: falta OPENAI_API_KEY")

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
	baseURL      string
	accounts     AccountService
	transactions TransactionService
	mcp          *mcp.Server
	tools        []map[string]any
	http         *http.Client
}

// NewChatService builds a ChatService. If apiKey is empty, the service returns
// ErrChatNotConfigured on every call.
func NewChatService(apiKey, model, baseURL string, accounts AccountService, transactions TransactionService) ChatService {
	mcpSrv := mcp.NewServer(accounts, transactions)
	return &chatService{
		apiKey:       apiKey,
		model:        model,
		baseURL:      baseURL,
		accounts:     accounts,
		transactions: transactions,
		mcp:          mcpSrv,
		tools:        openAITools(mcpSrv.Tools()),
		http:         &http.Client{Timeout: 90 * time.Second},
	}
}

func openAITools(tools []mcp.Tool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}
	return out
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
		msg, err := s.callOpenRouterRetry(ctx, messages)
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
					Message:              confirmationMessage(tc.Name, tc.Args),
					RequiresConfirmation: true,
					PendingAction:        map[string]any{"tool": tc.Name, "args": tc.Args},
				}, nil
			}
		}
		rawToolCalls := make([]map[string]any, 0, len(toolCalls))
		for _, tc := range toolCalls {
			rawToolCalls = append(rawToolCalls, map[string]any{
				"id": tc.ID, "type": "function",
				"function": map[string]any{"name": tc.Name, "arguments": tc.ArgsRaw},
			})
		}
		assistant := map[string]any{"role": "assistant", "tool_calls": rawToolCalls}
		messages = append(messages, assistant)
		for _, tc := range toolCalls {
			res := s.execute(ctx, userID, tc.Name, tc.Args)
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": tc.ID, "content": res})
		}
	}
	return &ChatResult{Message: "Lo siento, no pude completar la solicitud."}, nil
}

type toolCall struct {
	Name    string
	Args    map[string]any
	ArgsRaw string
	ID      string
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
		out = append(out, toolCall{Name: tc.Function.Name, Args: args, ArgsRaw: tc.Function.Arguments, ID: tc.ID})
	}
	return out
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

func (s *chatService) callOpenRouter(ctx context.Context, messages []map[string]any) (*openAIMessage, error) {
	body, _ := json.Marshal(map[string]any{"model": s.model, "messages": messages, "tools": s.tools})
	endpoint := strings.TrimRight(s.baseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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
	if len(data) == 0 {
		return nil, errors.New("ai provider returned an empty response")
	}
	var probe struct {
		Error *struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &probe) == nil && probe.Error != nil {
		return nil, fmt.Errorf("ai provider error (%s): %s", probe.Error.Type, probe.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("ai provider error: " + strings.TrimSpace(string(data)))
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

// callOpenRouterRetry invokes callOpenRouter, retrying transient provider errors.
func (s *chatService) callOpenRouterRetry(ctx context.Context, messages []map[string]any) (*openAIMessage, error) {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 700 * time.Millisecond):
			}
		}
		msg, err := s.callOpenRouter(ctx, messages)
		if err == nil {
			return msg, nil
		}
		lastErr = err
	}
	return nil, lastErr
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

// confirmationMessage builds a descriptive confirmation message for a critical action.
func confirmationMessage(tool string, args map[string]any) string {
	amount, _ := args["amount"].(string)
	switch tool {
	case "make_deposit":
		acct, _ := args["account_id"].(string)
		return fmt.Sprintf("Voy a depositar %s en la cuenta %s. ¿Confirmas?", amount, acct)
	case "make_withdrawal":
		acct, _ := args["account_id"].(string)
		return fmt.Sprintf("Voy a retirar %s de la cuenta %s. ¿Confirmas?", amount, acct)
	case "make_transfer":
		from, _ := args["from_account"].(string)
		to, _ := args["to_account"].(string)
		return fmt.Sprintf("Voy a transferir %s de la cuenta %s a la cuenta %s. ¿Confirmas?", amount, from, to)
	default:
		return "¿Confirmar la operación?"
	}
}

func (s *chatService) execute(ctx context.Context, userID, name string, args map[string]any) string {
	res, err := s.mcp.CallTool(ctx, userID, name, args)
	if err != nil {
		return `{"error": "` + err.Error() + `"}`
	}
	return res
}

func (s *chatService) executeAction(ctx context.Context, userID string, action map[string]any) (*ChatResult, error) {
	tool, _ := action["tool"].(string)
	if tool == "" {
		return &ChatResult{Message: "Acción no reconocida."}, nil
	}
	args, _ := action["args"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}
	if _, err := s.mcp.CallTool(ctx, userID, tool, args); err != nil {
		return &ChatResult{Message: "Error: " + err.Error()}, nil
	}
	return &ChatResult{Message: "Operación realizada correctamente."}, nil
}