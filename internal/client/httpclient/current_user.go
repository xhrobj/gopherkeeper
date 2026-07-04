package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const currentUserPath = "/api/v1/users/me"

// CurrentUser возвращает пользователя, связанного с текущим bearer token'ом.
func (c *Client) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+currentUserPath,
		nil,
	)
	if err != nil {
		return model.User{}, fmt.Errorf("create current user request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.User{}, fmt.Errorf("send current user request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return model.User{}, decodeAPIError(response)
	}

	var current userResponse
	if err := json.NewDecoder(response.Body).Decode(&current); err != nil {
		return model.User{}, fmt.Errorf("decode current user response: %w", err)
	}

	return model.User{
		ID:        current.ID,
		Login:     current.Login,
		CreatedAt: current.CreatedAt,
	}, nil
}
