package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/apierror"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// APIError представляет ошибку, возвращённую API Сервера.
type APIError struct {
	// StatusCode содержит HTTP-статус ответа.
	StatusCode int

	// Code содержит transport-neutral код ошибки API.
	Code apierror.Code

	// Message содержит текст ошибки, предназначенный для пользователя.
	Message string

	cause error
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type jsonRequest struct {
	operation      string
	method         string
	path           string
	accessToken    string
	headers        map[string]string
	requestBody    any
	expectedStatus int
	responseBody   any
}

// Error возвращает безопасное описание ошибки API.
func (e *APIError) Error() string {
	return fmt.Sprintf("api request failed: %s", e.Message)
}

// Unwrap возвращает transport-neutral причину ошибки API, если она известна Клиенту.
func (e *APIError) Unwrap() error {
	return e.cause
}

// FailureKind возвращает категорию ошибки для пользовательского интерфейса.
func (e *APIError) FailureKind() failure.Kind {
	return failure.KindFromAPIErrorCode(e.Code)
}

// UserMessage возвращает сообщение API, предназначенное для пользователя.
func (e *APIError) UserMessage() string {
	return e.Message
}

func (c *Client) doJSON(ctx context.Context, request jsonRequest) error {
	restyRequest := c.client.R().SetContext(ctx)
	if request.requestBody != nil {
		restyRequest.SetBody(request.requestBody)
	}
	if request.accessToken != "" {
		restyRequest.SetAuthToken(request.accessToken)
	}
	if len(request.headers) > 0 {
		restyRequest.SetHeaders(request.headers)
	}

	response, err := restyRequest.Execute(request.method, request.path)
	if err != nil {
		return failure.Network("send "+request.operation+" request", err)
	}

	if response.StatusCode() != request.expectedStatus {
		return decodeAPIError(response.StatusCode(), response.Status(), response.Body())
	}

	if request.responseBody == nil {
		return nil
	}
	if err := json.Unmarshal(response.Body(), request.responseBody); err != nil {
		return fmt.Errorf("decode %s response: %w", request.operation, err)
	}

	return nil
}

func decodeAPIError(statusCode int, status string, body []byte) error {
	var responseError errorResponse
	if err := json.Unmarshal(body, &responseError); err != nil {
		return fmt.Errorf("api request returned status %s", status)
	}

	if responseError.Code == "" || responseError.Message == "" {
		return fmt.Errorf("api request returned status %s", status)
	}

	code, known := apierror.Parse(responseError.Code)
	if !known {
		return fmt.Errorf("api request returned status %s with unknown error code %q", status, responseError.Code)
	}

	return &APIError{
		StatusCode: statusCode,
		Code:       code,
		Message:    responseError.Message,
		cause:      failure.CauseFromAPIErrorCode(code),
	}
}

func userFromResponse(response userResponse) model.User {
	return model.User{
		ID:        response.ID,
		Login:     response.Login,
		CreatedAt: response.CreatedAt,
	}
}
