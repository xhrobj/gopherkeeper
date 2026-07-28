package tui

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type backendStub struct {
	health func(context.Context) (string, error)

	register    func(context.Context, string, string) (string, error)
	login       func(context.Context, string, string) (string, error)
	currentUser func(context.Context) (string, error)
	logout      func(context.Context) error

	listRecords func(context.Context) ([]recordmodel.RecordMetadata, error)
	getRecord   func(context.Context, string) (recordmodel.Record, error)

	openCache  func(context.Context, string, string) ([]recordmodel.RecordMetadata, error)
	getCached  func(context.Context, string) (recordmodel.Record, error)
	closeCache func()

	createRecord func(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	updateRecord func(context.Context, string, int64, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	deleteRecord func(context.Context, string, int64) error

	sync func(context.Context, string) (SyncSummary, error)
}

type transportClosingBackendStub struct {
	backendStub
	closeTransport func() error
}

var _ Backend = backendStub{}

func (stub backendStub) Health(ctx context.Context) (string, error) {
	if stub.health == nil {
		return "", errors.New("unexpected Health call")
	}
	return stub.health(ctx)
}

func (stub backendStub) Register(
	ctx context.Context,
	login string,
	password string,
) (string, error) {
	if stub.register == nil {
		return "", errors.New("unexpected Register call")
	}
	return stub.register(ctx, login, password)
}

func (stub backendStub) Login(
	ctx context.Context,
	login string,
	password string,
) (string, error) {
	if stub.login == nil {
		return "", errors.New("unexpected Login call")
	}
	return stub.login(ctx, login, password)
}

func (stub backendStub) CurrentUser(ctx context.Context) (string, error) {
	if stub.currentUser == nil {
		return "", errors.New("unexpected CurrentUser call")
	}
	return stub.currentUser(ctx)
}

func (stub backendStub) Logout(ctx context.Context) error {
	if stub.logout == nil {
		return errors.New("unexpected Logout call")
	}
	return stub.logout(ctx)
}

func (stub backendStub) ListRecords(ctx context.Context) ([]recordmodel.RecordMetadata, error) {
	if stub.listRecords == nil {
		return nil, nil
	}
	return stub.listRecords(ctx)
}

func (stub backendStub) GetRecord(ctx context.Context, id string) (recordmodel.Record, error) {
	if stub.getRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected GetRecord call")
	}
	return stub.getRecord(ctx, id)
}

func (stub backendStub) OpenCache(
	ctx context.Context,
	login string,
	password string,
) ([]recordmodel.RecordMetadata, error) {
	if stub.openCache == nil {
		return nil, errors.New("unexpected OpenCache call")
	}
	return stub.openCache(ctx, login, password)
}

func (stub backendStub) GetCachedRecord(ctx context.Context, id string) (recordmodel.Record, error) {
	if stub.getCached == nil {
		return recordmodel.Record{}, errors.New("unexpected GetCachedRecord call")
	}
	return stub.getCached(ctx, id)
}

func (stub backendStub) CloseCache() {
	if stub.closeCache != nil {
		stub.closeCache()
	}
}

func (stub backendStub) CreateRecord(
	ctx context.Context,
	title string,
	payload recordmodel.RecordPayload,
) (recordmodel.Record, error) {
	if stub.createRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected CreateRecord call")
	}
	return stub.createRecord(ctx, title, payload)
}

func (stub backendStub) UpdateRecord(
	ctx context.Context,
	id string,
	revision int64,
	title string,
	payload recordmodel.RecordPayload,
) (recordmodel.Record, error) {
	if stub.updateRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected UpdateRecord call")
	}
	return stub.updateRecord(ctx, id, revision, title, payload)
}

func (stub backendStub) DeleteRecord(ctx context.Context, id string, revision int64) error {
	if stub.deleteRecord == nil {
		return errors.New("unexpected DeleteRecord call")
	}
	return stub.deleteRecord(ctx, id, revision)
}

func (stub backendStub) Sync(ctx context.Context, password string) (SyncSummary, error) {
	if stub.sync == nil {
		return SyncSummary{}, errors.New("unexpected Sync call")
	}
	return stub.sync(ctx, password)
}

func (stub *transportClosingBackendStub) CloseTransport() error {
	if stub.closeTransport == nil {
		return nil
	}

	return stub.closeTransport()
}

func staticBackendFactory(backend Backend) BackendFactory {
	return func(config.Config) (Backend, error) {
		return backend, nil
	}
}
