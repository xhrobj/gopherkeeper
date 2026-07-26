package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

func TestServerStatusCmd_ReturnsBackendResult(t *testing.T) {
	cmd := serverStatusCmd(
		context.Background(),
		backendStub{health: func(context.Context) (string, error) {
			return "ok", nil
		}},
		1,
		0,
	)

	result := commandResult[serverStatusResultMsg](t, cmd)
	if result.requestID != 1 || result.status != "ok" || result.err != nil {
		t.Fatalf("result = %#v, want request 1 and ok", result)
	}
}

func TestModel_ServerStatusUsesConfiguredBackend(t *testing.T) {
	cfg := config.Config{
		Address:    "vault.example:8443",
		CACertFile: "/tmp/ca.pem",
	}
	m := newTestModel(t, cfg, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.backend = backendStub{health: func(context.Context) (string, error) {
		return "ok", nil
	}}

	updated, cmd := m.activate(actionServerStatus)
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Server Status did not start health check")
	}
	if m.dialog != dialogServerStatus || m.statusState != serverStatusIdle {
		t.Fatalf("status dialog = %d, state = %d; want server status idle while request is pending", m.dialog, m.statusState)
	}
	if m.activeButton != 0 || !m.operations.request(operationServerStatus).pending || !m.interactionBlocked() {
		t.Fatalf(
			"checking state = button %d pending %t blocked %t, want Check/true/true",
			m.activeButton,
			m.operations.request(operationServerStatus).pending,
			m.interactionBlocked(),
		)
	}
	frame := m.spinnerFrameValue()
	assertViewContains(t, m.View().Content, "Server Status "+frame, "< Check >", "< OK >")
	assertViewContains(t, m.View().Content, "Checking...", "pending")
	assertViewExcludes(t, m.View().Content, "Please wait")

	result := commandResult[serverStatusResultMsg](t, cmd)
	updated, _ = m.Update(result)
	got := updated.(model)
	if got.statusState != serverStatusReady || got.statusValue != "ok" {
		t.Fatalf("status state = %d, value = %q; want ready ok", got.statusState, got.statusValue)
	}
	assertViewContains(t, got.View().Content,
		"Server Status",
		"Address",
		"vault.example:8443",
		"Status",
		"Available",
		"Health",
		"ok",
		"< Check >",
		"< OK >",
	)
}

func TestModel_ServerStatusRendersFoxProError(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
	m.width = 100
	m.height = 32
	m.backend = backendStub{health: func(context.Context) (string, error) {
		return "", errors.New("server unavailable: connection refused")
	}}

	updated, cmd := m.activate(actionServerStatus)
	m = updated.(model)
	result := commandResult[serverStatusResultMsg](t, cmd)
	updated, _ = m.Update(result)
	got := updated.(model)
	if got.statusState != serverStatusFailed {
		t.Fatalf("status state = %d, want failed", got.statusState)
	}
	plain := ansi.Strip(got.View().Content)
	for _, want := range []string{
		"Server Status",
		"localhost:8888",
		"< Check >",
		"< OK >",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("status view does not contain %q:\n%s", want, plain)
		}
	}
	assertServerStatusRow(t, plain, "Status", "Unreachable")
	assertServerStatusRow(t, plain, "Reason", "Connection refused")
}

func TestDescribeServerStatusError(t *testing.T) {
	tests := []struct {
		name       string
		message    string
		wantStatus string
		wantReason string
	}{
		{
			name:       "connection refused",
			message:    "send health request: Get \"https://localhost:8888/health\": dial tcp: connect: connection refused",
			wantStatus: "Unreachable",
			wantReason: "Connection refused",
		},
		{
			name:       "TLS certificate",
			message:    "send health request: Get \"https://localhost:8888/health\": tls: failed to verify certificate: x509: certificate signed by unknown authority",
			wantStatus: "TLS error",
			wantReason: "Certificate verification failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := describeServerStatusError(errors.New(test.message))
			if got.status != test.wantStatus || got.reason != test.wantReason {
				t.Fatalf("failure = %#v, want status %q reason %q", got, test.wantStatus, test.wantReason)
			}
		})
	}
}

func TestDescribeServerStatusError_UsesSafeTypedHealthMessage(t *testing.T) {
	err := failure.Wrap(
		failure.Unavailable,
		"health request returned status 503 Service Unavailable",
		"Server health check failed",
		nil,
	)

	got := describeServerStatusError(err)
	if got.status != "Unreachable" || got.reason != "Server health check failed" {
		t.Fatalf("failure = %#v, want status %q reason %q", got, "Unreachable", "Server health check failed")
	}
}

func TestModel_ServerStatusCheckShowsPendingState(t *testing.T) {
	tests := []struct {
		name         string
		state        serverStatusState
		value        string
		failure      serverStatusFailure
		wantContains []string
	}{
		{
			name:         "ready",
			state:        serverStatusReady,
			value:        "ok",
			wantContains: []string{"Available", "ok"},
		},
		{
			name:    "failed",
			state:   serverStatusFailed,
			failure: serverStatusFailure{status: "Unreachable", reason: "Connection refused"},
			wantContains: []string{
				"Unreachable",
				"Connection refused",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
			m.dialog = dialogServerStatus
			m.statusState = test.state
			m.statusValue = test.value
			m.statusFailure = test.failure
			m.activeButton = 0
			m.backend = backendStub{health: func(context.Context) (string, error) { return "ok", nil }}

			updated, cmd := m.activateDialogButton()
			got := updated.(model)
			if cmd == nil || !got.operations.request(operationServerStatus).pending {
				t.Fatal("Check did not start a new health check")
			}
			if got.statusState != test.state || got.statusValue != test.value || got.statusFailure != test.failure {
				t.Fatalf("previous result changed while pending: state=%d value=%q failure=%#v", got.statusState, got.statusValue, got.statusFailure)
			}
			view := got.View().Content
			assertViewContains(t, view, "Checking...", "pending")
			for _, previous := range test.wantContains {
				assertViewExcludes(t, view, previous)
			}
		})
	}
}

func TestRenderServerStatus_UsesSameWindowSizeAndNoBorder(t *testing.T) {
	theme := newTheme()
	ready := renderServerStatusWindow(theme, serverStatusWindowOptions{
		width: 58, address: "localhost:8888", state: serverStatusReady, health: "ok", activeButton: 1,
	})
	failed := renderServerStatusWindow(theme, serverStatusWindowOptions{
		width:        58,
		address:      "localhost:8888",
		state:        serverStatusFailed,
		failure:      serverStatusFailure{status: "Unreachable", reason: "Connection refused"},
		activeButton: 1,
	})
	if lipgloss.Width(ready) != lipgloss.Width(failed) || lipgloss.Height(ready) != lipgloss.Height(failed) {
		t.Fatalf(
			"status window sizes differ: ready=%dx%d failed=%dx%d",
			lipgloss.Width(ready),
			lipgloss.Height(ready),
			lipgloss.Width(failed),
			lipgloss.Height(failed),
		)
	}
	plain := ansi.Strip(ready)
	for _, border := range []string{"┌", "┐", "└", "┘", "│", "─"} {
		if strings.Contains(plain, border) {
			t.Fatalf("server status contains border character %q: %q", border, plain)
		}
	}
}

func TestModel_ServerStatusIgnoresStaleResult(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.operations.request(operationServerStatus).id = 2
	m.statusState = serverStatusReady
	m.statusValue = "ok"

	updated, _ := m.Update(serverStatusResultMsg{
		requestID: 1,
		status:    "stale",
	})
	got := updated.(model)
	if got.statusState != serverStatusReady || got.statusValue != "ok" {
		t.Fatalf("stale result changed state to %d, value %q", got.statusState, got.statusValue)
	}
}

func TestModel_ServerStatusUsesSpinnerAndGlobalBlockWhileChecking(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogServerStatus
	m.statusState = serverStatusReady
	m.statusValue = "ok"
	m.operations.request(operationServerStatus).pending = true
	m.activeButton = 0

	frame := m.spinnerFrameValue()
	assertViewContains(t, m.View().Content, "Server Status "+frame, "Checking...", "pending", "< Check >", "< OK >")
	assertViewExcludes(t, m.View().Content, "Available", "ok", "Please wait")

	for _, key := range []string{"tab", "shift+tab", "enter", "esc", "f10"} {
		updated, cmd := m.Update(keyPress(key))
		got := updated.(model)
		if cmd != nil {
			t.Fatalf("blocked key %q returned a command", key)
		}
		if got.dialog != dialogServerStatus || got.activeButton != 0 || !got.operations.request(operationServerStatus).pending {
			t.Fatalf(
				"blocked key %q changed status dialog: dialog %d button %d pending %t",
				key,
				got.dialog,
				got.activeButton,
				got.operations.request(operationServerStatus).pending,
			)
		}
	}
}

func TestStatusButtonsLayout_DisablesBothButtonsAndPreservesFocusWhileBlocked(t *testing.T) {
	theme := newTheme()
	rendered := statusButtonsLayout(
		statusButtonStyles{
			background:     theme.aboutBody,
			button:         theme.aboutButton,
			active:         theme.aboutButtonActive,
			disabled:       theme.aboutButtonDisabled,
			disabledActive: theme.aboutButtonDisabledActive,
		},
		40,
		0,
		true,
	).content

	if !strings.Contains(rendered, theme.aboutButtonDisabledActive.Render("< Check >")) {
		t.Fatalf("focused Check button is not rendered with disabled-active style: %q", ansi.Strip(rendered))
	}
	if !strings.Contains(rendered, theme.aboutButtonDisabled.Render("< OK >")) {
		t.Fatalf("OK button is not rendered with disabled style: %q", ansi.Strip(rendered))
	}
}

func TestNewModel_ServerStatusMinimumDurationIsHalfSecond(t *testing.T) {
	m := mustNewModel(t,
		context.Background(),
		config.Config{},
		"",
		buildinfo.Info{},
		staticBackendFactory(backendStub{}),
	)
	if m.statusMinDuration != 500*time.Millisecond {
		t.Fatalf("status minimum duration = %s, want 500ms", m.statusMinDuration)
	}
}

func TestServerStatusCmd_HonorsMinimumDuration(t *testing.T) {
	const minimum = 20 * time.Millisecond
	startedAt := time.Now()
	result := commandResult[serverStatusResultMsg](t, serverStatusCmd(
		context.Background(),
		backendStub{health: func(context.Context) (string, error) { return "ok", nil }},
		1,
		minimum,
	))
	if result.err != nil || result.status != "ok" {
		t.Fatalf("result = %#v, want ok", result)
	}
	if elapsed := time.Since(startedAt); elapsed < minimum {
		t.Fatalf("server status command returned after %s, want at least %s", elapsed, minimum)
	}
}

func assertServerStatusRow(t *testing.T, view, label, value string) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, label) && strings.Contains(line, value) {
			if strings.Index(line, value)-strings.Index(line, label) > 20 {
				t.Fatalf("%s value is shifted too far to the right: %q", label, line)
			}
			return
		}
	}
	t.Fatalf("status row %q = %q was not found:\n%s", label, value, view)
}
