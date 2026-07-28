package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_ConfigChangeRechecksSessionForNewRuntime(t *testing.T) {
	initial := config.Config{Address: "old.example:8443", CACertFile: "/tmp/ca.pem", SessionDir: "/tmp/old-session"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configAddress].setValue("new.example:9443")
	m.configForm.fields[configSessionDir].setValue("/tmp/new-session")
	m.configForm.focus = configSave

	var configured config.Config
	currentUserCalled := false
	m.backendFactory = func(cfg config.Config) (Backend, error) {
		configured = cfg
		return backendStub{
			currentUser: func(context.Context) (string, error) {
				currentUserCalled = true
				return "bob", nil
			},
		}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("config change did not start a session recheck")
	}
	if m.authentication.session.state != authUnknown || m.authentication.session.login != "" || m.dialog != dialogNone {
		t.Fatalf("state before recheck = auth %#v dialog %d", m.authentication.session, m.dialog)
	}

	updated, _ = m.Update(commandResult[currentUserResultMsg](t, cmd))
	got := updated.(model)
	if configured != got.config {
		t.Fatalf("configured runtime = %#v, want %#v", configured, got.config)
	}
	if !currentUserCalled {
		t.Fatal("CurrentUser was not called after runtime reconfiguration")
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" {
		t.Fatalf("state after recheck = %#v, want authenticated bob", got.authentication.session)
	}
}

func TestModel_ConfigChangeKeepsCurrentRuntimeWhenBackendCreationFails(t *testing.T) {
	initial := config.Config{Address: "old.example:8443", CACertFile: "/tmp/old-ca.pem"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configAddress].setValue("new.example:9443")
	m.configForm.fields[configCACertFile].setValue("/tmp/new-ca.pem")
	m.configForm.focus = configSave
	currentBackend := &backendStub{}
	m.backend = currentBackend
	want := errors.New("backend creation failed")
	m.backendFactory = func(config.Config) (Backend, error) {
		return nil, want
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)

	if cmd != nil {
		t.Fatal("failed runtime change started a command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want config", got.dialog)
	}
	if got.config != initial {
		t.Fatalf("config = %#v, want %#v", got.config, initial)
	}
	if got.backend != currentBackend {
		t.Fatal("backend changed after factory error")
	}
	const wantMessage = "Backend creation failed"
	if got.configForm.errorMessage != wantMessage {
		t.Fatalf("config error = %q, want %q", got.configForm.errorMessage, wantMessage)
	}
}

func TestNewModel_CreatesBackendForInitialRuntime(t *testing.T) {
	cfg := config.Config{
		Address:    "vault.example:8443",
		CACertFile: "/tmp/ca.pem",
		SessionDir: "/tmp/session",
		CacheDir:   "/tmp/cache",
	}

	var configured config.Config
	m := mustNewModel(t,
		context.Background(),
		cfg,
		"",
		buildinfo.Info{},
		func(got config.Config) (Backend, error) {
			configured = got
			return backendStub{
				currentUser: func(context.Context) (string, error) {
					return "", nil
				},
			}, nil
		},
	)
	m.cancelAllRequests()

	if configured != cfg {
		t.Fatalf("configured runtime = %#v, want %#v", configured, cfg)
	}
}

func TestModel_ConfigTransportChangePreservesOpenCacheWorkspace(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  " Alice ",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) {
			return "alice", nil
		}}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("transport change did not start session recheck")
	}
	if !m.recordFeature.workspace.open || m.recordFeature.workspace.source != recordSourceCache {
		t.Fatalf("cache workspace was closed during transport switch: %#v", m.recordFeature.workspace)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, cmd))
	got := updated.(model)
	if next != nil {
		t.Fatal("transport switch from an open cache started online record loading")
	}
	if !got.recordFeature.workspace.open || got.recordFeature.workspace.source != recordSourceCache {
		t.Fatalf("cache workspace was not preserved after session recheck: %#v", got.recordFeature.workspace)
	}
}

func TestModel_ConfigSessionChangeClosesCacheForDifferentUser(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/alice-session",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configSessionDir].setValue("/tmp/bob-session")
	m.configForm.focus = configSave

	closeCacheCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{
			currentUser: func(context.Context) (string, error) { return "bob", nil },
			closeCache:  func() { closeCacheCalls++ },
		}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("session directory change did not start session restore")
	}
	if !m.recordFeature.workspace.open || m.recordFeature.workspace.login != "alice" {
		t.Fatalf("cache workspace was closed before the new session was resolved: %#v", m.recordFeature.workspace)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if closeCacheCalls != 1 {
		t.Fatalf("cache close calls = %d, want 1", closeCacheCalls)
	}
	if !got.recordFeature.workspace.open || got.recordFeature.workspace.source != recordSourceServer {
		t.Fatalf("Bob server workspace was not opened after closing Alice cache: %#v", got.recordFeature.workspace)
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" {
		t.Fatalf("restored session = %#v, want authenticated bob", got.authentication.session)
	}
	if next == nil {
		t.Fatal("restoring another user did not start loading that user's server records")
	}
}

func TestModel_ConfigSessionChangeFailureClosesOpenCache(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/alice-session",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configSessionDir].setValue("/tmp/other-session")
	m.configForm.focus = configSave

	closeCacheCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{
			currentUser: func(context.Context) (string, error) {
				return "", errors.New("server unavailable")
			},
			closeCache: func() { closeCacheCalls++ },
		}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("session directory change did not start session restore")
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if next != nil {
		t.Fatal("failed session restore started another command")
	}
	if closeCacheCalls != 1 {
		t.Fatalf("cache close calls = %d, want 1", closeCacheCalls)
	}
	if got.recordFeature.workspace.open || got.recordFeature.workspace.source == recordSourceCache {
		t.Fatalf("cache remained open after failed restore from another session directory: %#v", got.recordFeature.workspace)
	}
	if got.authentication.session.state != authGuest {
		t.Fatalf("session state after failed restore = %d, want guest", got.authentication.session.state)
	}
}

func TestModel_ConfigTransportFailurePreservesCurrentSession(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/session",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) {
			return "", errors.New("gRPC unavailable")
		}}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("transport change did not start session check")
	}
	if m.authentication.currentUserCheck != currentUserCheckReconfigure {
		t.Fatalf("current user check mode = %d, want reconfigure", m.authentication.currentUserCheck)
	}
	if !m.authentication.session.authenticated() || m.authentication.session.login != "alice" {
		t.Fatalf("session was cleared before transport check: %#v", m.authentication.session)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if next != nil {
		t.Fatal("failed transport session check started another command")
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" {
		t.Fatalf("session after transport failure = %#v, want authenticated alice", got.authentication.session)
	}
	if got.alert != alertError || got.alertTitle != "Session check failed" || got.alertReturnDialog != dialogNone {
		t.Fatalf("alert after transport failure = state %d title %q return %d", got.alert, got.alertTitle, got.alertReturnDialog)
	}
}

func TestRuntimeConfigChanged(t *testing.T) {
	https := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "https.example:8443",
		GRPCAddress: "grpc.example:9443",
		CACertFile:  "/tmp/ca.pem",
		SessionDir:  "/tmp/session",
		CacheDir:    "/tmp/cache",
	}
	grpc := https
	grpc.Transport = config.TransportGRPC

	tests := []struct {
		name      string
		previous  config.Config
		candidate config.Config
		want      bool
	}{
		{name: "unchanged HTTPS", previous: https, candidate: https},
		{name: "switch transport", previous: https, candidate: grpc, want: true},
		{name: "HTTPS active address", previous: https, candidate: withConfigAddress(https, "other.example:8443"), want: true},
		{name: "HTTPS inactive gRPC address", previous: https, candidate: withConfigGRPCAddress(https, "other.example:9443")},
		{name: "gRPC active address", previous: grpc, candidate: withConfigGRPCAddress(grpc, "other.example:9443"), want: true},
		{name: "gRPC inactive HTTPS address", previous: grpc, candidate: withConfigAddress(grpc, "other.example:8443")},
		{name: "CA certificate", previous: https, candidate: withConfigCACertFile(https, "/tmp/other-ca.pem"), want: true},
		{name: "session directory", previous: https, candidate: withConfigSessionDir(https, "/tmp/other-session"), want: true},
		{name: "cache directory", previous: https, candidate: withConfigCacheDir(https, "/tmp/other-cache"), want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := runtimeConfigChanged(test.previous, test.candidate); got != test.want {
				t.Fatalf("runtimeConfigChanged() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestModel_ConfigInactiveAddressChangeDoesNotReplaceRuntime(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/session",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	closeTransportCalls := 0
	currentBackend := &transportClosingBackendStub{
		backendStub: backendStub{},
		closeTransport: func() error {
			closeTransportCalls++
			return nil
		},
	}
	m.backend = currentBackend
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.grpcAddress.setValue("grpc.example:9443")
	m.configForm.focus = configSave
	factoryCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		factoryCalls++
		return backendStub{}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	got := updated.(model)
	if command != nil {
		t.Fatal("inactive gRPC address change started a session check")
	}
	if factoryCalls != 0 {
		t.Fatalf("backend factory calls = %d, want 0", factoryCalls)
	}
	if closeTransportCalls != 0 {
		t.Fatalf("active HTTPS transport close calls = %d, want 0", closeTransportCalls)
	}
	if got.backend != currentBackend {
		t.Fatal("active runtime was replaced after changing inactive gRPC address")
	}
	if got.config.GRPCAddress != "grpc.example:9443" {
		t.Fatalf("saved gRPC address = %q", got.config.GRPCAddress)
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" {
		t.Fatalf("session changed after inactive address edit: %#v", got.authentication.session)
	}
}

func TestModel_ConfigChangeClosesPreviousTransport(t *testing.T) {
	initial := config.Config{Transport: config.TransportHTTPS, Address: "localhost:8888", GRPCAddress: "localhost:9090"}
	m := newTestModel(t, initial, buildinfo.Info{})
	oldCloseCalls := 0
	oldBackend := &transportClosingBackendStub{
		backendStub: backendStub{},
		closeTransport: func() error {
			oldCloseCalls++
			return nil
		},
	}
	m.backend = oldBackend
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) { return "alice", nil }}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)
	if cmd == nil {
		t.Fatal("config change did not start session recheck")
	}
	if oldCloseCalls != 1 {
		t.Fatalf("previous transport close calls = %d, want 1", oldCloseCalls)
	}
	if got.backend == oldBackend {
		t.Fatal("backend was not replaced")
	}
}

func TestModel_ConfigSaveFailureClosesPreparedTransportAndKeepsCurrentBackend(t *testing.T) {
	initial := config.Config{Transport: config.TransportHTTPS, Address: "localhost:8888", GRPCAddress: "localhost:9090"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.configFile = "/tmp/client.json"
	m.saveConfig = func(string, config.Config) error { return errors.New("write failed") }
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	currentBackend := &backendStub{}
	m.backend = currentBackend
	preparedCloseCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return &transportClosingBackendStub{
			backendStub: backendStub{},
			closeTransport: func() error {
				preparedCloseCalls++
				return nil
			},
		}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)
	if cmd != nil {
		t.Fatal("failed config save started a command")
	}
	if preparedCloseCalls != 1 {
		t.Fatalf("prepared transport close calls = %d, want 1", preparedCloseCalls)
	}
	if got.backend != currentBackend || got.config != initial {
		t.Fatalf("runtime changed after save failure: backend changed=%t config=%#v", got.backend != currentBackend, got.config)
	}
	if got.configForm.errorMessage != "Unable to save config file" {
		t.Fatalf("config error = %q", got.configForm.errorMessage)
	}
}
