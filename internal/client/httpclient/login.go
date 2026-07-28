package httpclient

import (
	"context"
	"net/http"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const loginPath = "/api/v1/auth/login"

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        userResponse `json:"user"`
}

// Login аутентифицирует пользователя на Сервере и возвращает bearer token.
func (c *Client) Login(ctx context.Context, login, password string) (model.Authentication, error) {
	var loggedIn loginResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:      "login",
		method:         http.MethodPost,
		path:           loginPath,
		requestBody:    loginRequest{Login: login, Password: password},
		expectedStatus: http.StatusOK,
		responseBody:   &loggedIn,
	}); err != nil {
		return model.Authentication{}, err
	}

	return model.Authentication{
		AccessToken: loggedIn.AccessToken,
		ExpiresAt:   loggedIn.ExpiresAt,
		User:        userFromResponse(loggedIn.User),
	}, nil
}
