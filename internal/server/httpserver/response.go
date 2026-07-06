package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/server/httperror"
)

type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

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
