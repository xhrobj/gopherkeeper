package grpcclient

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/grpc/metadata"
)

const authorizationMetadataKey = "authorization"

// Register регистрирует нового пользователя на Сервере.
func (c *Client) Register(ctx context.Context, login, password string) (model.User, error) {
	request := &gopherkeeperpb.RegisterRequest{}
	request.SetLogin(login)
	request.SetPassword(password)

	callCtx, cancel := withRequestTimeout(ctx)
	defer cancel()

	response, err := c.auth.Register(callCtx, request)
	if err != nil {
		return model.User{}, mapRPCError("registration", err)
	}
	if response == nil || response.GetUser() == nil {
		return model.User{}, invalidResponseError("registration", errors.New("user is missing"))
	}

	return userFromProto(response.GetUser())
}

// Login аутентифицирует пользователя на Сервере и возвращает bearer token.
func (c *Client) Login(ctx context.Context, login, password string) (model.Authentication, error) {
	request := &gopherkeeperpb.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(password)

	callCtx, cancel := withRequestTimeout(ctx)
	defer cancel()

	response, err := c.auth.Login(callCtx, request)
	if err != nil {
		return model.Authentication{}, mapRPCError("login", err)
	}
	if response == nil || response.GetUser() == nil || response.GetAccessToken() == "" {
		return model.Authentication{}, invalidResponseError("login", errors.New("authentication data is incomplete"))
	}

	user, err := userFromProto(response.GetUser())
	if err != nil {
		return model.Authentication{}, invalidResponseError("login", err)
	}
	expiresAt, err := timeFromProto(response.GetExpiresAt())
	if err != nil {
		return model.Authentication{}, invalidResponseError("login", err)
	}

	return model.Authentication{
		AccessToken: response.GetAccessToken(),
		ExpiresAt:   expiresAt,
		User:        user,
	}, nil
}

// CurrentUser возвращает пользователя, связанного с текущим bearer token'ом.
func (c *Client) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.auth.CurrentUser(callCtx, &gopherkeeperpb.CurrentUserRequest{})
	if err != nil {
		return model.User{}, mapRPCError("current user", err)
	}
	if response == nil || response.GetUser() == nil {
		return model.User{}, invalidResponseError("current user", errors.New("user is missing"))
	}

	return userFromProto(response.GetUser())
}

func withBearerToken(ctx context.Context, accessToken string) context.Context {
	if accessToken == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(
		ctx,
		authorizationMetadataKey,
		"Bearer "+accessToken,
	)
}
