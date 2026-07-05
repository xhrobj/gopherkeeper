package httpserver

import (
	"encoding/json"
	"net/http"
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
	writeJSONResponse(w, statusCode, errorResponse{
		Code:    code,
		Message: message,
	})
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		return
	}
}
