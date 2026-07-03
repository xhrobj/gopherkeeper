package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const registerPath = "/api/v1/auth/register"

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

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Register регистрирует нового пользователя на Сервере.
func (c *Client) Register(ctx context.Context, login, password string) (model.User, error) {
	requestBody, err := json.Marshal(registerRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return model.User{}, fmt.Errorf("encode registration request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+registerPath,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create registration request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.User{}, fmt.Errorf("send registration request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusCreated {
		return model.User{}, decodeAPIError(response)
	}

	var registered registerResponse
	if err := json.NewDecoder(response.Body).Decode(&registered); err != nil {
		return model.User{}, fmt.Errorf("decode registration response: %w", err)
	}

	return model.User{
		ID:        registered.ID,
		Login:     registered.Login,
		CreatedAt: registered.CreatedAt,
	}, nil
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
