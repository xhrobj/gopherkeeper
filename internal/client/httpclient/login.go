package httpclient

import (
	"context"
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

// Login аутентифицирует пользователя на Сервере и возвращает bearer token.
func (c *Client) Login(ctx context.Context, login, password string) (LoginResult, error) {
	var loggedIn loginResponse
	if err := c.doJSON(ctx, jsonRequest{
		operation:      "login",
		method:         http.MethodPost,
		path:           loginPath,
		requestBody:    loginRequest{Login: login, Password: password},
		expectedStatus: http.StatusOK,
		responseBody:   &loggedIn,
	}); err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		AccessToken: loggedIn.AccessToken,
		TokenType:   loggedIn.TokenType,
		ExpiresAt:   loggedIn.ExpiresAt,
		User:        userFromResponse(loggedIn.User),
	}, nil
}
