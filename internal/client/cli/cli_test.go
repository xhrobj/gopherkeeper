package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

var testBuildInfo = buildinfo.Info{
	Version: "v0.4.2",
	Date:    "2026-06-30",
	Commit:  "deadbeef",
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
	if runners.whoami == nil {
		runners.whoami = unexpectedWhoamiRunner(t)
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

func unexpectedWhoamiRunner(t *testing.T) outputRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Helper()
		t.Fatal("whoami command must not run")
		return nil
	}
}
