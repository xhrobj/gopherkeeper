package grpcserver

import (
	"context"

	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
)

type authService struct {
	gopherkeeperpb.UnimplementedAuthServiceServer
	registerer UserRegisterer
	auth       UserAuthenticator
	users      CurrentUserReader
}

func newAuthService(deps Dependencies) *authService {
	return &authService{
		registerer: deps.Registerer,
		auth:       deps.Authenticator,
		users:      deps.CurrentUserReader,
	}
}

func (service *authService) Register(
	ctx context.Context,
	request *gopherkeeperpb.RegisterRequest,
) (*gopherkeeperpb.RegisterResponse, error) {
	if request == nil {
		return nil, transportError(errInvalidRequest)
	}

	if service.registerer == nil {
		return nil, transportError(errInvalidDependencies)
	}

	user, err := service.registerer.Register(ctx, request.GetLogin(), request.GetPassword())
	if err != nil {
		return nil, transportError(err)
	}

	responseUser, err := newProtoUser(user)
	if err != nil {
		return nil, transportError(err)
	}

	response := &gopherkeeperpb.RegisterResponse{}
	response.SetUser(responseUser)

	return response, nil
}

func (service *authService) Login(
	ctx context.Context,
	request *gopherkeeperpb.LoginRequest,
) (*gopherkeeperpb.LoginResponse, error) {
	if request == nil {
		return nil, transportError(errInvalidRequest)
	}

	if service.auth == nil {
		return nil, transportError(errInvalidDependencies)
	}

	result, err := service.auth.Authenticate(ctx, request.GetLogin(), request.GetPassword())
	if err != nil {
		return nil, transportError(err)
	}

	responseUser, err := newProtoUser(result.User)
	if err != nil {
		return nil, transportError(err)
	}
	expiresAt, err := newProtoTimestamp(result.ExpiresAt)
	if err != nil {
		return nil, transportError(err)
	}

	response := &gopherkeeperpb.LoginResponse{}
	response.SetUser(responseUser)
	response.SetAccessToken(result.AccessToken)
	response.SetExpiresAt(expiresAt)

	return response, nil
}

func (service *authService) CurrentUser(
	ctx context.Context,
	request *gopherkeeperpb.CurrentUserRequest,
) (*gopherkeeperpb.CurrentUserResponse, error) {
	if request == nil {
		return nil, transportError(errInvalidRequest)
	}

	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if service.users == nil {
		return nil, transportError(errInvalidDependencies)
	}

	user, err := service.users.FindByID(ctx, userID)
	if err != nil {
		return nil, transportError(err)
	}

	responseUser, err := newProtoUser(user)
	if err != nil {
		return nil, transportError(err)
	}

	response := &gopherkeeperpb.CurrentUserResponse{}
	response.SetUser(responseUser)

	return response, nil
}
