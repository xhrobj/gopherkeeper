package httpclient

import (
	"context"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const registerPath = "/api/v1/auth/register"

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerResponse = userResponse

// Register регистрирует нового пользователя на Сервере.
func (c *Client) Register(ctx context.Context, login, password string) (model.User, error) {
	var registered registerResponse
	if err := c.doJSON(ctx, jsonRequest{
		operation:      "registration",
		method:         http.MethodPost,
		path:           registerPath,
		requestBody:    registerRequest{Login: login, Password: password},
		expectedStatus: http.StatusCreated,
		responseBody:   &registered,
	}); err != nil {
		return model.User{}, err
	}

	return userFromResponse(registered), nil
}
