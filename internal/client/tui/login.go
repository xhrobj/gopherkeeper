package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	domainmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type loginResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func loginCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	userName string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		canonicalLogin, err := backend.Login(ctx, userName, password)
		return loginResultMsg{requestID: requestID, login: canonicalLogin, err: err}
	}
}

func cleanLoginError(err error) string {
	if err == nil {
		return "Unknown login error"
	}

	if errors.Is(err, context.Canceled) {
		return "Login canceled"
	}

	if errors.Is(err, domainmodel.ErrInvalidCredentials) {
		return "Invalid login or password"
	}

	message := strings.TrimSpace(err.Error())
	lower := strings.ToLower(message)

	if strings.Contains(lower, "invalid login or password") || strings.Contains(lower, "invalid credentials") {
		return "Invalid login or password"
	}

	failure := describeServerStatusError(err)
	if failure.status != "Connection error" {
		return failure.reason
	}

	for _, prefix := range []string{"login user: ", "create client application: "} {
		message = strings.TrimPrefix(message, prefix)
	}

	if message == "" {
		return "Unable to log in"
	}

	return capitalizeFirst(message)
}
