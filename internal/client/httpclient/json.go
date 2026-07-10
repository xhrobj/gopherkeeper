package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	body, err := encodeRequestBody(request.operation, request.requestBody)
	if err != nil {
		return err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, request.method, c.baseURL+request.path, body)
	if err != nil {
		return fmt.Errorf("create %s request: %w", request.operation, err)
	}
	if request.requestBody != nil {
		httpRequest.Header.Set("Content-Type", "application/json")
	}
	if request.accessToken != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.accessToken)
	}
	for name, value := range request.headers {
		httpRequest.Header.Set(name, value)
	}

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("send %s request: %w", request.operation, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != request.expectedStatus {
		return decodeAPIError(response)
	}

	if request.responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(request.responseBody); err != nil {
		return fmt.Errorf("decode %s response: %w", request.operation, err)
	}

	return nil
}

func encodeRequestBody(operation string, requestBody any) (io.Reader, error) {
	if requestBody == nil {
		return nil, nil
	}

	data, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("encode %s request: %w", operation, err)
	}

	return bytes.NewReader(data), nil
}

func decodeAPIError(response *http.Response) error {
	var responseError errorResponse
	if err := json.NewDecoder(response.Body).Decode(&responseError); err != nil {
		return fmt.Errorf("api request returned status %s", response.Status)
	}

	if responseError.Code == "" || responseError.Message == "" {
		return fmt.Errorf("api request returned status %s", response.Status)
	}

	return &APIError{
		StatusCode: response.StatusCode,
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
