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
	if err := c.doBearerJSON(
		ctx,
		"current user",
		http.MethodGet,
		currentUserPath,
		accessToken,
		nil,
		http.StatusOK,
		&current,
	); err != nil {
		return model.User{}, err
	}

	return userFromResponse(current), nil
}
