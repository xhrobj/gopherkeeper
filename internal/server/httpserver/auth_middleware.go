package httpserver

import (
	"context"
	"net/http"
	"strings"
)

const authorizationSchemeBearer = "Bearer"

type userIDContextKey struct{}

// TokenValidator проверяет bearer token и возвращает идентификатор пользователя.
type TokenValidator interface {
	// Validate проверяет bearer token.
	Validate(ctx context.Context, token string) (int64, error)
}

// WithAuthentication проверяет bearer token и добавляет user ID в контекст запроса.
func WithAuthentication(handler http.Handler, validator TokenValidator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeUnauthorizedResponse(w)
			return
		}

		userID, err := validator.Validate(r.Context(), token)
		if err != nil {
			writeUnauthorizedResponse(w)
			return
		}

		ctx := context.WithValue(r.Context(), userIDContextKey{}, userID)
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext возвращает идентификатор аутентифицированного пользователя.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey{}).(int64)
	return userID, ok
}

func bearerToken(headerValue string) (string, bool) {
	fields := strings.Fields(headerValue)
	if len(fields) != 2 || !strings.EqualFold(fields[0], authorizationSchemeBearer) {
		return "", false
	}

	return fields[1], true
}

func writeUnauthorizedResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", authorizationSchemeBearer)
	writeErrorResponse(
		w,
		http.StatusUnauthorized,
		errorCodeUnauthorized,
		errorMessageUnauthorized,
	)
}
