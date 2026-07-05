package httpclient

import (
	"context"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const currentUserPath = "/api/v1/users/me"

// CurrentUser возвращает пользователя, связанного с текущим bearer token'ом.
func (c *Client) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
	var current userResponse
	if err := c.doJSON(ctx, jsonRequest{
		operation:      "current user",
		method:         http.MethodGet,
		path:           currentUserPath,
		accessToken:    accessToken,
		expectedStatus: http.StatusOK,
		responseBody:   &current,
	}); err != nil {
		return model.User{}, err
	}

	return userFromResponse(current), nil
}
