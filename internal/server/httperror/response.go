package httperror

import (
	"encoding/json"
	"net/http"
)

// Response содержит безопасное описание HTTP-ошибки API.
type Response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Write записывает JSON-ошибку с указанным HTTP-статусом.
func Write(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(Response{
		Code:    code,
		Message: message,
	}); err != nil {
		return
	}
}
