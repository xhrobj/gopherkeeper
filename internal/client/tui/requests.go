package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

type requestState struct {
	pending bool
	id      uint64
	cancel  context.CancelFunc
}

func (request *requestState) begin(parent context.Context) (context.Context, uint64) {
	request.cancelRequest()
	request.pending = true
	request.id++

	ctx, cancel := context.WithCancel(parent)
	request.cancel = cancel

	return ctx, request.id
}

func (request *requestState) cancelRequest() {
	if request.cancel != nil {
		request.cancel()
		request.cancel = nil
	}

	if request.pending {
		request.id++
	}

	request.pending = false
}

func (request *requestState) finish() {
	if request.cancel != nil {
		request.cancel()
		request.cancel = nil
	}

	request.pending = false
}

func (request requestState) accepts(id uint64) bool {
	return request.pending && id == request.id
}

func (m *model) beginCurrentUserCheck() tea.Cmd {
	m.logoutRequest.cancelRequest()
	m.auth = authSession{state: authUnknown}

	requestCtx, requestID := m.authRequest.begin(m.ctx)
	return currentUserCommand(requestCtx, m.backend, requestID)
}

func (m *model) cancelAuthRequest() {
	m.authRequest.cancelRequest()
}

func (m *model) cancelLoginRequest() {
	m.loginRequest.cancelRequest()
}

func (m *model) cancelRegisterRequest() {
	m.registerRequest.cancelRequest()
}

func (m *model) cancelLogoutRequest() {
	m.logoutRequest.cancelRequest()
}

func (m *model) cancelStatusRequest() {
	m.statusRequest.cancelRequest()
}

func (m *model) cancelAllRequests() {
	m.cancelAuthRequest()
	m.cancelLoginRequest()
	m.cancelRegisterRequest()
	m.cancelLogoutRequest()
	m.cancelStatusRequest()
}
