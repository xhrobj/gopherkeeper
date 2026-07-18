package cli

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestTUICommand_RunsWithResolvedConfiguration(t *testing.T) {
	isolateClientConfig(t)

	wantConfig := config.Config{
		Address:     "localhost:8443",
		CACertFile:  "ca.pem",
		SessionFile: "session.json",
		CacheDir:    "cache",
	}
	var gotConfig config.Config
	var gotConfigFile string
	var gotInfo buildinfo.Info
	var gotInput io.Reader
	var gotOutput io.Writer

	runner := tuiRunnerStub{run: func(
		_ context.Context,
		cfg config.Config,
		configFile string,
		info buildinfo.Info,
		input io.Reader,
		output io.Writer,
	) error {
		gotConfig = cfg
		gotConfigFile = configFile
		gotInfo = info
		gotInput = input
		gotOutput = output
		return nil
	}}

	input := bytes.NewBufferString("")
	var output bytes.Buffer
	if err := run(context.Background(), []string{
		"gopherkeeper",
		"--address", wantConfig.Address,
		"--ca-cert", wantConfig.CACertFile,
		"--session-file", wantConfig.SessionFile,
		"--cache-dir", wantConfig.CacheDir,
		"tui",
	}, runOptions{
		input:       input,
		output:      &output,
		errorOutput: io.Discard,
		info:        testBuildInfo,
		factory:     newClientFactoryStub(t),
		passwords:   streamPasswordReader{},
		tui:         runner,
	}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !reflect.DeepEqual(gotConfig, wantConfig) {
		t.Errorf("config = %+v, want %+v", gotConfig, wantConfig)
	}
	if gotConfigFile != "" {
		t.Errorf("config file = %q, want empty", gotConfigFile)
	}
	if gotInfo != testBuildInfo {
		t.Errorf("build info = %+v, want %+v", gotInfo, testBuildInfo)
	}
	if gotInput != input {
		t.Error("input was not passed to TUI runner")
	}
	if gotOutput != &output {
		t.Error("output was not passed to TUI runner")
	}
}

func TestTUICommand_PassesConfigFilePath(t *testing.T) {
	isolateClientConfig(t)

	configFile := writeClientConfig(t, `{
  "address": "localhost:9443",
  "ca_cert_file": "ca.pem",
  "session_file": "session.json",
  "cache_dir": "cache"
}`)

	var gotConfigFile string
	runner := tuiRunnerStub{run: func(
		_ context.Context,
		_ config.Config,
		path string,
		_ buildinfo.Info,
		_ io.Reader,
		_ io.Writer,
	) error {
		gotConfigFile = path
		return nil
	}}

	if err := run(context.Background(), []string{
		"gopherkeeper",
		"--config", configFile,
		"tui",
	}, runOptions{
		input:       strings.NewReader(""),
		output:      io.Discard,
		errorOutput: io.Discard,
		info:        testBuildInfo,
		factory:     newClientFactoryStub(t),
		passwords:   streamPasswordReader{},
		tui:         runner,
	}); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if gotConfigFile != configFile {
		t.Fatalf("config file = %q, want %q", gotConfigFile, configFile)
	}
}

func TestNewTUIBackendBuildsIndependentRuntime(t *testing.T) {
	firstConfig := config.Config{
		Address:     "localhost:8080",
		SessionFile: t.TempDir() + "/first-session.json",
	}
	firstBackend := newTUIBackend(firstConfig).(*tuiBackend)

	if firstBackend.healthClientError != nil ||
		firstBackend.applicationError != nil ||
		firstBackend.logoutError != nil {
		t.Fatalf(
			"initial runtime errors = health %v application %v logout %v",
			firstBackend.healthClientError,
			firstBackend.applicationError,
			firstBackend.logoutError,
		)
	}
	if firstBackend.healthClient == nil ||
		firstBackend.application == nil ||
		firstBackend.logoutApplication == nil {
		t.Fatal("initial runtime contains nil dependencies")
	}

	secondConfig := config.Config{
		Address:     "localhost:9090",
		SessionFile: t.TempDir() + "/second-session.json",
	}
	secondBackend := newTUIBackend(secondConfig).(*tuiBackend)

	if secondBackend == firstBackend {
		t.Fatal("backend factory reused a mutable runtime")
	}
	if secondBackend.healthClient == firstBackend.healthClient {
		t.Fatal("health client was reused across runtime configurations")
	}
	if secondBackend.application == firstBackend.application {
		t.Fatal("application runtime was reused across configurations")
	}
	if secondBackend.logoutApplication == firstBackend.logoutApplication {
		t.Fatal("logout runtime was reused across configurations")
	}
}
