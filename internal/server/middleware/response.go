package middleware

import (
	"encoding/json"
	"net/http"
)

const (
	errorCodeUnauthorized    = "unauthorized"
	errorMessageUnauthorized = "missing or invalid bearer token"
)

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeErrorResponse(
	w http.ResponseWriter,
	statusCode int,
	code string,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(errorResponse{
		Code:    code,
		Message: message,
	}); err != nil {
		return
	}
}
