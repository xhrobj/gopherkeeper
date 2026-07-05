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

// APIError представляет безопасную ошибку, возвращённую API Сервера.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
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

func (c *Client) doJSON(
	ctx context.Context,
	operation string,
	method string,
	path string,
	requestBody any,
	expectedStatus int,
	responseBody any,
) error {
	return c.doJSONWithAuthorization(
		ctx,
		operation,
		method,
		path,
		"",
		requestBody,
		expectedStatus,
		responseBody,
	)
}

func (c *Client) doBearerJSON(
	ctx context.Context,
	operation string,
	method string,
	path string,
	accessToken string,
	requestBody any,
	expectedStatus int,
	responseBody any,
) error {
	return c.doJSONWithAuthorization(
		ctx,
		operation,
		method,
		path,
		accessToken,
		requestBody,
		expectedStatus,
		responseBody,
	)
}

func (c *Client) doJSONWithAuthorization(
	ctx context.Context,
	operation string,
	method string,
	path string,
	accessToken string,
	requestBody any,
	expectedStatus int,
	responseBody any,
) error {
	body, err := encodeRequestBody(operation, requestBody)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create %s request: %w", operation, err)
	}
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send %s request: %w", operation, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != expectedStatus {
		return decodeAPIError(response)
	}

	if responseBody == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode %s response: %w", operation, err)
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
