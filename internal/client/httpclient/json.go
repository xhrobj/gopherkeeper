package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// APIError представляет ошибку, возвращённую API Сервера.
type APIError struct {
	// StatusCode содержит HTTP-статус ответа.
	StatusCode int

	// Code содержит код ошибки API.
	Code string

	// Message содержит текст ошибки, предназначенный для пользователя.
	Message string
}

// Error возвращает безопасное описание ошибки API.
func (e *APIError) Error() string {
	return fmt.Sprintf("api request failed: %s", e.Message)
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
		return fmt.Errorf("send %s request: %w", request.operation, err)
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

	return &APIError{
		StatusCode: statusCode,
		Code:       responseError.Code,
		Message:    responseError.Message,
	}
}

func userFromResponse(response userResponse) model.User {
	return model.User{
		ID:        response.ID,
		Login:     response.Login,
		CreatedAt: response.CreatedAt,
	}
}
