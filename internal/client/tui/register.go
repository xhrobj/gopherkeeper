package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	domainmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type registerResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func registerCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	login string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		canonicalLogin, err := backend.Register(ctx, login, password)
		return registerResultMsg{requestID: requestID, login: canonicalLogin, err: err}
	}
}

func cleanRegisterError(err error) string {
	if err == nil {
		return "Unknown registration error"
	}

	if errors.Is(err, context.Canceled) {
		return "Registration canceled"
	}

	if errors.Is(err, domainmodel.ErrLoginAlreadyExists) {
		return strings.TrimSpace(err.Error())
	}

	message := strings.TrimSpace(err.Error())
	failure := describeServerStatusError(err)

	if failure.status != "Connection error" {
		return failure.reason
	}

	for _, prefix := range []string{"register user: ", "create client application: "} {
		message = strings.TrimPrefix(message, prefix)
	}

	if message == "" {
		return "Unable to register"
	}

	return capitalizeFirst(message)
}
