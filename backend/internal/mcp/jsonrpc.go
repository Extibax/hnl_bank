package mcp

import (
	"context"
	"encoding/json"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HandleJSONRPC processes a single MCP JSON-RPC 2.0 message over stdio and
// returns the response bytes (nil for notifications).
func (s *Server) HandleJSONRPC(req []byte) ([]byte, error) {
	var r rpcRequest
	if err := json.Unmarshal(req, &r); err != nil {
		return s.errorResponse(nil, -32700, "Parse error: "+err.Error()), nil
	}
	switch r.Method {
	case "initialize":
		return s.okResponse(r.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "hnl-bank-mcp", "version": "1.0.0"},
		}), nil
	case "tools/list":
		return s.okResponse(r.ID, map[string]any{"tools": s.Tools()}), nil
	case "tools/call":
		return s.handleToolCall(r.ID, r.Params), nil
	case "ping":
		return s.okResponse(r.ID, map[string]any{}), nil
	case "notifications/initialized":
		return nil, nil
	default:
		return s.errorResponse(r.ID, -32601, "Method not found: "+r.Method), nil
	}
}

func (s *Server) handleToolCall(id json.RawMessage, params json.RawMessage) []byte {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return s.errorResponse(id, -32602, "Invalid params: "+err.Error())
	}
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}
	// The authenticated user id is passed as part of the arguments for this demo.
	userID, _ := p.Arguments["user_id"].(string)
	delete(p.Arguments, "user_id")
	text, err := s.CallTool(context.Background(), userID, p.Name, p.Arguments)
	if err != nil {
		return s.okResponse(id, map[string]any{"content": []map[string]any{{"type": "text", "text": "Error: " + err.Error()}}, "isError": true})
	}
	return s.okResponse(id, map[string]any{"content": []map[string]any{{"type": "text", "text": text}}})
}

func (s *Server) okResponse(id json.RawMessage, result any) []byte {
	b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: id, Result: result})
	return b
}

func (s *Server) errorResponse(id json.RawMessage, code int, msg string) []byte {
	b, _ := json.Marshal(rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
	return b
}