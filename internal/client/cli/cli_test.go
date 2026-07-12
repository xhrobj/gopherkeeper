package cli

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type applicationStub struct {
	register     func(context.Context, string, string) (model.User, error)
	login        func(context.Context, string, string) (model.User, error)
	whoami       func(context.Context) (model.User, error)
	createRecord func(context.Context, usecase.CreateRecordRequest) (model.Record, error)
	updateRecord func(context.Context, usecase.UpdateRecordRequest) (model.Record, error)
	listRecords  func(context.Context) ([]model.RecordMetadata, error)
	getRecord    func(context.Context, string) (model.Record, error)
	deleteRecord func(context.Context, usecase.DeleteRecordRequest) error
}

func newApplicationStub(t *testing.T) *applicationStub {
	t.Helper()

	return &applicationStub{
		register: func(context.Context, string, string) (model.User, error) {
			t.Helper()
			t.Fatal("Register must not be called")
			return model.User{}, nil
		},
		login: func(context.Context, string, string) (model.User, error) {
			t.Helper()
			t.Fatal("Login must not be called")
			return model.User{}, nil
		},
		whoami: func(context.Context) (model.User, error) {
			t.Helper()
			t.Fatal("Whoami must not be called")
			return model.User{}, nil
		},
		createRecord: func(context.Context, usecase.CreateRecordRequest) (model.Record, error) {
			t.Helper()
			t.Fatal("CreateRecord must not be called")
			return model.Record{}, nil
		},
		updateRecord: func(context.Context, usecase.UpdateRecordRequest) (model.Record, error) {
			t.Helper()
			t.Fatal("UpdateRecord must not be called")
			return model.Record{}, nil
		},
		listRecords: func(context.Context) ([]model.RecordMetadata, error) {
			t.Helper()
			t.Fatal("ListRecords must not be called")
			return nil, nil
		},
		getRecord: func(context.Context, string) (model.Record, error) {
			t.Helper()
			t.Fatal("GetRecord must not be called")
			return model.Record{}, nil
		},
		deleteRecord: func(context.Context, usecase.DeleteRecordRequest) error {
			t.Helper()
			t.Fatal("DeleteRecord must not be called")
			return nil
		},
	}
}

func (s *applicationStub) Register(ctx context.Context, login, password string) (model.User, error) {
	return s.register(ctx, login, password)
}

func (s *applicationStub) Login(ctx context.Context, login, password string) (model.User, error) {
	return s.login(ctx, login, password)
}

func (s *applicationStub) Whoami(ctx context.Context) (model.User, error) {
	return s.whoami(ctx)
}

func (s *applicationStub) CreateRecord(
	ctx context.Context,
	request usecase.CreateRecordRequest,
) (model.Record, error) {
	return s.createRecord(ctx, request)
}

func (s *applicationStub) UpdateRecord(
	ctx context.Context,
	request usecase.UpdateRecordRequest,
) (model.Record, error) {
	return s.updateRecord(ctx, request)
}

func (s *applicationStub) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {
	return s.listRecords(ctx)
}

func (s *applicationStub) GetRecord(ctx context.Context, recordID string) (model.Record, error) {
	return s.getRecord(ctx, recordID)
}

func (s *applicationStub) DeleteRecord(ctx context.Context, request usecase.DeleteRecordRequest) error {
	return s.deleteRecord(ctx, request)
}

type logoutApplicationStub struct {
	logout func(context.Context) error
}

func (s logoutApplicationStub) Logout(ctx context.Context) error {
	return s.logout(ctx)
}

type healthClientStub struct {
	health func(context.Context) (string, error)
}

func (s healthClientStub) Health(ctx context.Context) (string, error) {
	return s.health(ctx)
}

type clientFactoryStub struct {
	newApplication       func(config.Config) (application, error)
	newLogoutApplication func(config.Config) (logoutApplication, error)
	newHealthClient      func(config.Config) (healthClient, error)
}

func newClientFactoryStub(t *testing.T) *clientFactoryStub {
	t.Helper()

	return &clientFactoryStub{
		newApplication: func(config.Config) (application, error) {
			t.Helper()
			t.Fatal("application factory must not be called")
			return nil, nil
		},
		newLogoutApplication: func(config.Config) (logoutApplication, error) {
			t.Helper()
			t.Fatal("logout application factory must not be called")
			return nil, nil
		},
		newHealthClient: func(config.Config) (healthClient, error) {
			t.Helper()
			t.Fatal("health client factory must not be called")
			return nil, nil
		},
	}
}

func (s *clientFactoryStub) NewApplication(cfg config.Config) (application, error) {
	return s.newApplication(cfg)
}

func (s *clientFactoryStub) NewLogoutApplication(cfg config.Config) (logoutApplication, error) {
	return s.newLogoutApplication(cfg)
}

func (s *clientFactoryStub) NewHealthClient(cfg config.Config) (healthClient, error) {
	return s.newHealthClient(cfg)
}

var testBuildInfo = buildinfo.Info{
	Version: "v0.4.2",
	Date:    "2026-06-30",
	Commit:  "deadbeef",
}

func isolateClientConfig(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("CONFIG", "")
	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")
	t.Setenv("SESSION_FILE", "")
}

func runTestCommand(
	t *testing.T,
	args []string,
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	factory clientFactory,
) error {
	t.Helper()

	if input == nil {
		input = strings.NewReader("")
	}
	if factory == nil {
		factory = newClientFactoryStub(t)
	}

	return run(context.Background(), args, runOptions{
		input:       input,
		output:      output,
		errorOutput: errorOutput,
		info:        testBuildInfo,
		factory:     factory,
	})
}

func recordCreatorFunc(
	fn func(context.Context, usecase.CreateRecordRequest) (model.Record, error),
) application {
	return &applicationStub{createRecord: fn}
}

func recordUpdaterFunc(
	fn func(context.Context, usecase.UpdateRecordRequest) (model.Record, error),
) application {
	return &applicationStub{updateRecord: fn}
}

func recordGetterFunc(
	fn func(context.Context, string) (model.Record, error),
) application {
	return &applicationStub{getRecord: fn}
}

func cardRecordGetterFunc(
	fn func(context.Context, string) (model.Record, error),
) application {
	return &applicationStub{getRecord: fn}
}

func userRegistererFunc(
	fn func(context.Context, string, string) (model.User, error),
) application {
	return &applicationStub{register: fn}
}

func userLoggerFunc(
	fn func(context.Context, string, string) (model.User, error),
) application {
	return &applicationStub{login: fn}
}

func currentUserGetterFunc(
	fn func(context.Context) (model.User, error),
) application {
	return &applicationStub{whoami: fn}
}

func userLogoutterFunc(fn func(context.Context) error) logoutApplication {
	return logoutApplicationStub{logout: fn}
}
