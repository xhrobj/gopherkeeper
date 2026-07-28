package grpcserver

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

type databasePingerFunc func(context.Context) error

type tokenValidatorFunc func(context.Context, string) (int64, error)

type userRegistererFunc func(context.Context, string, string) (model.User, error)

type userAuthenticatorFunc func(context.Context, string, string) (service.AuthenticationResult, error)

type currentUserReaderFunc func(context.Context, int64) (model.User, error)

type recordManagerStub struct {
	create func(context.Context, service.CreateRecordRequest) (model.Record, error)
	list   func(context.Context, int64) ([]model.RecordMetadata, error)
	get    func(context.Context, int64, string) (model.Record, error)
	update func(context.Context, service.UpdateRecordRequest) (model.Record, error)
	delete func(context.Context, service.DeleteRecordRequest) error
}

func (fn databasePingerFunc) Ping(ctx context.Context) error {
	return fn(ctx)
}

func (fn tokenValidatorFunc) Validate(ctx context.Context, token string) (int64, error) {
	return fn(ctx, token)
}

func (fn userRegistererFunc) Register(ctx context.Context, login, password string) (model.User, error) {
	return fn(ctx, login, password)
}

func (fn userAuthenticatorFunc) Authenticate(
	ctx context.Context,
	login, password string,
) (service.AuthenticationResult, error) {
	return fn(ctx, login, password)
}

func (fn currentUserReaderFunc) FindByID(ctx context.Context, id int64) (model.User, error) {
	return fn(ctx, id)
}

func (stub recordManagerStub) Create(
	ctx context.Context,
	request service.CreateRecordRequest,
) (model.Record, error) {
	return stub.create(ctx, request)
}

func (stub recordManagerStub) List(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	return stub.list(ctx, userID)
}

func (stub recordManagerStub) Get(
	ctx context.Context,
	userID int64,
	recordID string,
) (model.Record, error) {
	return stub.get(ctx, userID, recordID)
}

func (stub recordManagerStub) Update(
	ctx context.Context,
	request service.UpdateRecordRequest,
) (model.Record, error) {
	return stub.update(ctx, request)
}

func (stub recordManagerStub) Delete(ctx context.Context, request service.DeleteRecordRequest) error {
	return stub.delete(ctx, request)
}

func completeDependencies() Dependencies {
	return Dependencies{
		Database: databasePingerFunc(func(context.Context) error {
			return nil
		}),
		Registerer: userRegistererFunc(func(context.Context, string, string) (model.User, error) {
			return model.User{}, nil
		}),
		Authenticator: userAuthenticatorFunc(func(context.Context, string, string) (service.AuthenticationResult, error) {
			return service.AuthenticationResult{}, nil
		}),
		TokenValidator: tokenValidatorFunc(func(context.Context, string) (int64, error) {
			return 1, nil
		}),
		CurrentUserReader: currentUserReaderFunc(func(context.Context, int64) (model.User, error) {
			return model.User{}, nil
		}),
		Records: recordManagerStub{
			create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
				return model.Record{}, nil
			},
			list: func(context.Context, int64) ([]model.RecordMetadata, error) {
				return nil, nil
			},
			get: func(context.Context, int64, string) (model.Record, error) {
				return model.Record{}, nil
			},
			update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
				return model.Record{}, nil
			},
			delete: func(context.Context, service.DeleteRecordRequest) error {
				return nil
			},
		},
	}
}

func newProtoRegisterRequest(login, password string) *gopherkeeperpb.RegisterRequest {
	request := &gopherkeeperpb.RegisterRequest{}
	request.SetLogin(login)
	request.SetPassword(password)
	return request
}

func newProtoLoginRequest(login, password string) *gopherkeeperpb.LoginRequest {
	request := &gopherkeeperpb.LoginRequest{}
	request.SetLogin(login)
	request.SetPassword(password)
	return request
}

func newProtoTextPayload(text, metadata string) *gopherkeeperpb.RecordPayload {
	value := &gopherkeeperpb.TextPayload{}
	value.SetText(text)
	value.SetMetadata(metadata)

	payload := &gopherkeeperpb.RecordPayload{}
	payload.SetText(value)
	return payload
}

func newProtoBinaryPayload(filename string, data []byte, setData bool) *gopherkeeperpb.RecordPayload {
	value := &gopherkeeperpb.BinaryPayload{}
	value.SetFilename(filename)
	if setData {
		value.SetData(data)
	}

	payload := &gopherkeeperpb.RecordPayload{}
	payload.SetBinary(value)
	return payload
}
