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
)

type recordCreatorFunc func(context.Context, usecase.CreateRecordRequest) (usecase.Record, error)

func (f recordCreatorFunc) CreateRecord(
	ctx context.Context,
	request usecase.CreateRecordRequest,
) (usecase.Record, error) {
	return f(ctx, request)
}

type recordUpdaterFunc func(context.Context, usecase.UpdateRecordRequest) (usecase.Record, error)

func (f recordUpdaterFunc) UpdateRecord(
	ctx context.Context,
	request usecase.UpdateRecordRequest,
) (usecase.Record, error) {
	return f(ctx, request)
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
}

func runTestCommand(
	t *testing.T,
	args []string,
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	runners commandRunners,
) error {
	t.Helper()

	if input == nil {
		input = strings.NewReader("")
	}
	if runners.health == nil {
		runners.health = unexpectedHealthRunner(t)
	}
	if runners.register == nil {
		runners.register = unexpectedRegisterRunner(t)
	}
	if runners.login == nil {
		runners.login = unexpectedLoginRunner(t)
	}
	if runners.logout == nil {
		runners.logout = unexpectedLogoutRunner(t)
	}
	if runners.whoami == nil {
		runners.whoami = unexpectedWhoamiRunner(t)
	}
	if runners.createTextRecord == nil {
		runners.createTextRecord = unexpectedCreateTextRecordRunner(t)
	}
	if runners.createCredentialsRecord == nil {
		runners.createCredentialsRecord = unexpectedCreateCredentialsRecordRunner(t)
	}
	if runners.createCardRecord == nil {
		runners.createCardRecord = unexpectedCreateCardRecordRunner(t)
	}
	if runners.updateTextRecord == nil {
		runners.updateTextRecord = unexpectedUpdateTextRecordRunner(t)
	}
	if runners.updateCredentialsRecord == nil {
		runners.updateCredentialsRecord = unexpectedUpdateCredentialsRecordRunner(t)
	}
	if runners.updateCardRecord == nil {
		runners.updateCardRecord = unexpectedUpdateCardRecordRunner(t)
	}
	if runners.listRecords == nil {
		runners.listRecords = unexpectedListRecordsRunner(t)
	}
	if runners.getRecord == nil {
		runners.getRecord = unexpectedGetRecordRunner(t)
	}
	if runners.deleteRecord == nil {
		runners.deleteRecord = unexpectedDeleteRecordRunner(t)
	}

	return run(context.Background(), args, runOptions{
		input:       input,
		output:      output,
		errorOutput: errorOutput,
		info:        testBuildInfo,
		runners:     runners,
	})
}

func unexpectedHealthRunner(t *testing.T) outputRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Helper()
		t.Fatal("health command must not run")
		return nil
	}
}

func unexpectedRegisterRunner(t *testing.T) passwordRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Reader, io.Writer, io.Writer, string, bool) error {
		t.Helper()
		t.Fatal("register command must not run")
		return nil
	}
}

func unexpectedLoginRunner(t *testing.T) passwordRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Reader, io.Writer, io.Writer, string, bool) error {
		t.Helper()
		t.Fatal("login command must not run")
		return nil
	}
}

func unexpectedLogoutRunner(t *testing.T) outputRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Helper()
		t.Fatal("logout command must not run")
		return nil
	}
}

func unexpectedWhoamiRunner(t *testing.T) outputRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Helper()
		t.Fatal("whoami command must not run")
		return nil
	}
}

func unexpectedCreateTextRecordRunner(t *testing.T) textRecordCreateRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer, string, string, string) error {
		t.Helper()
		t.Fatal("records create-text command must not run")
		return nil
	}
}

func unexpectedUpdateTextRecordRunner(t *testing.T) textRecordUpdateRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer, textRecordUpdateCommandRequest) error {
		t.Helper()
		t.Fatal("records update-text command must not run")
		return nil
	}
}

func unexpectedCreateCredentialsRecordRunner(t *testing.T) credentialsRecordCreateRunner {
	t.Helper()

	return func(
		context.Context,
		config.Config,
		io.Reader,
		io.Writer,
		io.Writer,
		credentialsRecordCreateCommandRequest,
	) error {
		t.Helper()
		t.Fatal("records create-credentials command must not run")
		return nil
	}
}

func unexpectedUpdateCredentialsRecordRunner(t *testing.T) credentialsRecordUpdateRunner {
	t.Helper()

	return func(
		context.Context,
		config.Config,
		io.Reader,
		io.Writer,
		io.Writer,
		credentialsRecordUpdateCommandRequest,
	) error {
		t.Helper()
		t.Fatal("records update-credentials command must not run")
		return nil
	}
}

func unexpectedCreateCardRecordRunner(t *testing.T) cardRecordCreateRunner {
	t.Helper()

	return func(
		context.Context,
		config.Config,
		io.Reader,
		io.Writer,
		io.Writer,
		cardRecordCreateCommandRequest,
	) error {
		t.Helper()
		t.Fatal("records create-card command must not run")
		return nil
	}
}

func unexpectedUpdateCardRecordRunner(t *testing.T) cardRecordUpdateRunner {
	t.Helper()

	return func(
		context.Context,
		config.Config,
		io.Reader,
		io.Writer,
		io.Writer,
		cardRecordUpdateCommandRequest,
	) error {
		t.Helper()
		t.Fatal("records update-card command must not run")
		return nil
	}
}

func unexpectedListRecordsRunner(t *testing.T) outputRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Helper()
		t.Fatal("records list command must not run")
		return nil
	}
}

func unexpectedGetRecordRunner(t *testing.T) recordGetRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer, string) error {
		t.Helper()
		t.Fatal("records get command must not run")
		return nil
	}
}

func unexpectedDeleteRecordRunner(t *testing.T) recordDeleteRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer, string, int64) error {
		t.Helper()
		t.Fatal("records delete command must not run")
		return nil
	}
}
