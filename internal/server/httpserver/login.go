package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const tokenTypeBearer = "Bearer"

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        userResponse `json:"user"`
}

func loginHandler(authenticator UserAuthenticator) http.HandlerFunc {
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

		request, err := decodeLoginRequest(r)
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
				errorMessageInvalidLoginRequest,
			)
			return
		}

		result, err := authenticator.Authenticate(
			r.Context(),
			request.Login,
			request.Password,
		)
		if err != nil {
			writeLoginError(w, err)
			return
		}

		writeNoStoreHeaders(w)
		writeJSONResponse(w, http.StatusOK, loginResponse{
			AccessToken: result.AccessToken,
			TokenType:   tokenTypeBearer,
			ExpiresAt:   result.ExpiresAt.UTC(),
			User: userResponse{
				ID:        result.User.ID,
				Login:     result.User.Login,
				CreatedAt: result.User.CreatedAt,
			},
		})
	}
}

func decodeLoginRequest(r *http.Request) (loginRequest, error) {
	var request loginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		return loginRequest{}, err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return loginRequest{}, errMultipleJSONValues
		}

		return loginRequest{}, err
	}

	return request, nil
}

func writeLoginError(w http.ResponseWriter, err error) {
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeErrorResponse(
			w,
			http.StatusUnauthorized,
			errorCodeInvalidCredentials,
			errorMessageInvalidCredentials,
		)
		return
	}

	writeErrorResponse(
		w,
		http.StatusInternalServerError,
		errorCodeInternal,
		errorMessageInternal,
	)
}

func writeNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}
