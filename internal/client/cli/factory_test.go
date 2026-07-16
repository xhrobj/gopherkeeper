package cli

import (
	"errors"
	"testing"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestApplicationFromCommand(t *testing.T) {
	wantConfig := config.Config{Address: "localhost:8080"}
	wantApplication := newApplicationStub(t)
	factory := newClientFactoryStub(t)
	factory.newApplication = func(got config.Config) (application, error) {
		if got != wantConfig {
			t.Errorf("NewApplication() config = %+v, want %+v", got, wantConfig)
		}
		return wantApplication, nil
	}
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: wantConfig,
	}}

	got, err := applicationFromCommand(command, factory)
	if err != nil {
		t.Fatalf("applicationFromCommand() error = %v", err)
	}
	if got != wantApplication {
		t.Errorf("applicationFromCommand() = %T, want application stub", got)
	}
}

func TestApplicationFromCommand_ReturnsConfigError(t *testing.T) {
	factory := newClientFactoryStub(t)
	command := &urfavecli.Command{Metadata: map[string]any{}}

	_, err := applicationFromCommand(command, factory)
	if err == nil || err.Error() != "client config is missing" {
		t.Fatalf("applicationFromCommand() error = %v, want missing config error", err)
	}
}

func TestApplicationFromCommand_ReturnsFactoryError(t *testing.T) {
	factoryErr := errors.New("factory failed")
	factory := newClientFactoryStub(t)
	factory.newApplication = func(config.Config) (application, error) {
		return nil, factoryErr
	}
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: config.Config{Address: "localhost:8080"},
	}}

	_, err := applicationFromCommand(command, factory)
	if !errors.Is(err, factoryErr) {
		t.Fatalf("applicationFromCommand() error = %v, want factory error", err)
	}
}

func TestOfflineApplicationFromCommand(t *testing.T) {
	wantConfig := config.Config{
		Address:     "localhost:8080",
		CACertFile:  "missing-ca.pem",
		SessionFile: "missing-session.json",
		CacheDir:    "cache",
	}
	wantApplication := newApplicationStub(t)
	factory := newClientFactoryStub(t)
	factory.newOfflineApplication = func(got config.Config) (application, error) {
		if got != wantConfig {
			t.Errorf("NewOfflineApplication() config = %+v, want %+v", got, wantConfig)
		}
		return wantApplication, nil
	}
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: wantConfig,
	}}

	got, err := offlineApplicationFromCommand(command, factory)
	if err != nil {
		t.Fatalf("offlineApplicationFromCommand() error = %v", err)
	}
	if got != wantApplication {
		t.Errorf("offlineApplicationFromCommand() = %T, want application stub", got)
	}
}

func TestOfflineApplicationFromCommand_ReturnsFactoryError(t *testing.T) {
	factoryErr := errors.New("offline factory failed")
	factory := newClientFactoryStub(t)
	factory.newOfflineApplication = func(config.Config) (application, error) {
		return nil, factoryErr
	}
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: config.Config{Address: "localhost:8080"},
	}}

	_, err := offlineApplicationFromCommand(command, factory)
	if !errors.Is(err, factoryErr) {
		t.Fatalf("offlineApplicationFromCommand() error = %v, want %v", err, factoryErr)
	}
}
