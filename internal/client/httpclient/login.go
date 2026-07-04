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

const loginPath = "/api/v1/auth/login"

// LoginResult содержит результат успешной аутентификации пользователя.
type LoginResult struct {
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
	User        model.User
}

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

type userResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

// Login аутентифицирует пользователя на Сервере и возвращает bearer token.
func (c *Client) Login(ctx context.Context, login, password string) (LoginResult, error) {
	requestBody, err := json.Marshal(loginRequest{
		Login:    login,
		Password: password,
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("encode login request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+loginPath,
		bytes.NewReader(requestBody),
	)
	if err != nil {
		return LoginResult{}, fmt.Errorf("create login request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return LoginResult{}, fmt.Errorf("send login request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return LoginResult{}, decodeAPIError(response)
	}

	var loggedIn loginResponse
	if err := json.NewDecoder(response.Body).Decode(&loggedIn); err != nil {
		return LoginResult{}, fmt.Errorf("decode login response: %w", err)
	}

	return LoginResult{
		AccessToken: loggedIn.AccessToken,
		TokenType:   loggedIn.TokenType,
		ExpiresAt:   loggedIn.ExpiresAt,
		User: model.User{
			ID:        loggedIn.User.ID,
			Login:     loggedIn.User.Login,
			CreatedAt: loggedIn.User.CreatedAt,
		},
	}, nil
}
