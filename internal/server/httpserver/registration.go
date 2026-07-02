package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const maxRequestBodySize int64 = 4 * 1024 * 1024

var errMultipleJSONValues = errors.New("request body must contain one JSON value")

const (
	errorCodeInvalidRequest       = "invalid_request"
	errorCodeLoginAlreadyExists   = "login_already_exists"
	errorCodePayloadTooLarge      = "payload_too_large"
	errorCodeUnsupportedMediaType = "unsupported_media_type"
	errorCodeInternal             = "internal_error"
)

const (
	errorMessageInvalidRequest       = "invalid registration data"
	errorMessageLoginAlreadyExists   = "login is already registered"
	errorMessagePayloadTooLarge      = "request body is too large"
	errorMessageUnsupportedMediaType = "content type must be application/json"
	errorMessageInternal             = "internal server error"
)

type userRegistrar interface {
	Register(
		ctx context.Context,
		login string,
		password string,
	) (model.User, error)
}

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func registerHandler(registrar userRegistrar) http.HandlerFunc {
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

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		defer func() {
			_ = r.Body.Close()
		}()

		request, err := decodeRegisterRequest(r)
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

		user, err := registrar.Register(
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

func decodeRegisterRequest(r *http.Request) (registerRequest, error) {
	var request registerRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return registerRequest{}, err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return registerRequest{}, errMultipleJSONValues
		}

		return registerRequest{}, err
	}

	return request, nil
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
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
