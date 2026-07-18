package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

type authState int

const (
	authUnknown authState = iota
	authGuest
	authAuthenticated
)

type authSession struct {
	state authState
	login string
}

func (session authSession) authenticated() bool {
	return session.state == authAuthenticated
}

type currentUserResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func currentUserCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
) tea.Cmd {
	return func() tea.Msg {
		login, err := backend.CurrentUser(ctx)
		return currentUserResultMsg{requestID: requestID, login: login, err: err}
	}
}

func isNotLoggedIn(err error) bool {
	return errors.Is(err, usecase.ErrNotLoggedIn)
}

func cleanCurrentUserError(err error) string {
	if err == nil {
		return "Unknown session error"
	}
	if errors.Is(err, context.Canceled) {
		return "Session check canceled"
	}

	message := strings.TrimSpace(err.Error())
	failure := describeServerStatusError(err)
	if failure.status != "Connection error" {
		return failure.reason
	}

	for _, prefix := range []string{
		"get current user: ",
		"load online session: ",
		"create client application: ",
	} {
		message = strings.TrimPrefix(message, prefix)
	}
	if message == "" {
		return "Unable to check the current session"
	}

	return capitalizeFirst(message)
}
