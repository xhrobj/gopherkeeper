package httpserver

import (
	"encoding/json"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/server/httperror"
)

func writeErrorResponse(w http.ResponseWriter, statusCode int, code string, message string) {
	httperror.Write(w, statusCode, code, message)
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		return
	}
}
