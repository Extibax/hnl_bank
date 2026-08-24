package response

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the standard error envelope returned by the API.
type ErrorBody struct {
	Message string `json:"message"` 
	Code    string `json:"code"` 
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

// Error writes an error envelope with the given status code and code string.
func Error(w http.ResponseWriter, status int, message, code string) {
	JSON(w, status, map[string]any{
		"error": ErrorBody{Message: message, Code: code},
	})
}
