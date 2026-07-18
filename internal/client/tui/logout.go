package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type logoutResultMsg struct {
	requestID uint64
	err       error
}

func logoutCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
) tea.Cmd {
	return func() tea.Msg {
		return logoutResultMsg{requestID: requestID, err: backend.Logout(ctx)}
	}
}

func cleanLogoutError(err error) string {
	if err == nil {
		return "Unknown logout error"
	}

	if errors.Is(err, context.Canceled) {
		return "Logout canceled"
	}

	message := strings.TrimSpace(strings.TrimPrefix(err.Error(), "delete online session: "))
	if message == "" {
		return "Unable to log out"
	}

	return capitalizeFirst(message)
}
