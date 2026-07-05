package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	errorCodeInvalidRequest       = "invalid_request"
	errorCodeInvalidCredentials   = "invalid_credentials"
	errorCodeLoginAlreadyExists   = "login_already_exists"
	errorCodePayloadTooLarge      = "payload_too_large"
	errorCodeUnsupportedMediaType = "unsupported_media_type"
	errorCodeInternal             = "internal_error"
)

const (
	errorMessageInvalidRequest       = "invalid registration data"
	errorMessageInvalidLoginRequest  = "invalid login request"
	errorMessageInvalidCredentials   = "invalid login or password"
	errorMessageLoginAlreadyExists   = "login is already registered"
	errorMessagePayloadTooLarge      = "request body is too large"
	errorMessageUnsupportedMediaType = "content type must be application/json"
	errorMessageInternal             = "internal server error"
)

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
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
				errorMessageInvalidRequest,
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
			errorMessageInvalidRequest,
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
