package httpserver

import (
	"errors"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
)

func currentUserHandler(users CurrentUserReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		user, err := users.FindByID(r.Context(), userID)
		if err != nil {
			writeCurrentUserError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusOK, userResponse{
			ID:        user.ID,
			Login:     user.Login,
			CreatedAt: user.CreatedAt,
		})
	})
}

func writeCurrentUserError(w http.ResponseWriter, err error) {
	if errors.Is(err, model.ErrUserNotFound) {
		middleware.WriteUnauthorizedResponse(w)
		return
	}

	writeErrorResponse(
		w,
		http.StatusInternalServerError,
		errorCodeInternal,
		errorMessageInternal,
	)
}
