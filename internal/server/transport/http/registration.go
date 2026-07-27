package httpserver

import (
	"errors"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	errorMessageInvalidRegistrationRequest = "invalid registration data"
	errorMessageLoginAlreadyExists         = "login is already registered"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func registerHandler(registerer UserRegisterer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeErrorResponse(
				w,
				http.StatusUnsupportedMediaType,
				errorCodeUnsupportedMediaType,
				errorMessageUnsupportedMediaType,
			)
			return
		}

		request, err := decodeRegisterRequest(w, r)
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeErrorResponse(
					w,
					http.StatusRequestEntityTooLarge,
					errorCodePayloadTooLarge,
					errorMessagePayloadTooLarge,
				)
				return
			}

			writeErrorResponse(
				w,
				http.StatusBadRequest,
				errorCodeInvalidRequest,
				errorMessageInvalidRegistrationRequest,
			)
			return
		}

		user, err := registerer.Register(
			r.Context(),
			request.Login,
			request.Password,
		)
		if err != nil {
			writeRegistrationError(w, err)
			return
		}

		writeJSONResponse(w, http.StatusCreated, userResponse{
			ID:        user.ID,
			Login:     user.Login,
			CreatedAt: user.CreatedAt,
		})
	}
}

func decodeRegisterRequest(w http.ResponseWriter, r *http.Request) (registerRequest, error) {
	var request registerRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		return registerRequest{}, err
	}

	return request, nil
}

func writeRegistrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidLogin),
		errors.Is(err, service.ErrInvalidPassword),
		errors.Is(err, service.ErrPasswordTooShort),
		errors.Is(err, service.ErrPasswordTooLong):
		writeErrorResponse(
			w,
			http.StatusBadRequest,
			errorCodeInvalidRequest,
			errorMessageInvalidRegistrationRequest,
		)

	case errors.Is(err, model.ErrLoginAlreadyExists):
		writeErrorResponse(
			w,
			http.StatusConflict,
			errorCodeLoginAlreadyExists,
			errorMessageLoginAlreadyExists,
		)

	default:
		writeErrorResponse(
			w,
			http.StatusInternalServerError,
			errorCodeInternal,
			errorMessageInternal,
		)
	}
}
