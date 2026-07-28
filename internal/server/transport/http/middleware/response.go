package middleware

import (
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/server/transport/http/httperror"
)

const (
	errorCodeUnauthorized    = string(apierror.Unauthorized)
	errorMessageUnauthorized = "missing or invalid bearer token"
)

func writeErrorResponse(w http.ResponseWriter, statusCode int, code string, message string) {
	httperror.Write(w, statusCode, code, message)
}
