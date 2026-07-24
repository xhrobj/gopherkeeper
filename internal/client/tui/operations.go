package tui

import (
	"context"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type operationKind uint8

const (
	operationNone operationKind = iota
	operationCurrentUser
	operationLogin
	operationRegister
	operationLogout
	operationServerStatus
	operationListRecords
	operationViewRecord
	operationLoadRecordForEdit
	operationCreateRecord
	operationEditRecord
	operationDeleteRecord
	operationBinarySave
	operationCount
)

type requestState struct {
	pending bool
	id      uint64
	cancel  context.CancelFunc
}

type operationCoordinator struct {
	requests [operationCount]requestState
}

func (operations *operationCoordinator) begin(parent context.Context, kind operationKind) (context.Context, uint64) {
	request := operations.request(kind)
	request.cancelRequest()
	request.pending = true
	request.id++

	ctx, cancel := context.WithCancel(parent)
	request.cancel = cancel

	return ctx, request.id
}

func (operations *operationCoordinator) cancel(kind operationKind) {
	operations.request(kind).cancelRequest()
}

func (operations *operationCoordinator) cancelAll() {
	for kind := operationKind(1); kind < operationCount; kind++ {
		operations.cancel(kind)
	}
}

func (operations *operationCoordinator) finish(kind operationKind) {
	operations.request(kind).finish()
}

func (operations *operationCoordinator) accepts(kind operationKind, id uint64) bool {
	return operations.request(kind).accepts(id)
}

func (operations *operationCoordinator) pending(kind operationKind) bool {
	return operations.request(kind).pending
}

func (operations *operationCoordinator) request(kind operationKind) *requestState {
	if kind <= operationNone || kind >= operationCount {
		panic("invalid TUI operation kind")
	}
	return &operations.requests[kind]
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

func (m *model) beginCurrentUserCheck(mode currentUserCheckMode) tea.Cmd {
	m.operations.cancel(operationLogout)
	m.authentication.currentUserCheck = mode
	if mode == currentUserCheckRestore {
		m.authentication.session = authSession{state: authUnknown}
	}

	requestCtx, requestID := m.operations.begin(m.ctx, operationCurrentUser)
	return m.operationCommand(
		operationCurrentUser,
		currentUserCommand(requestCtx, m.backend, requestID),
	)
}

func (m *model) cancelRecordEditRequests() {
	m.operations.cancel(operationLoadRecordForEdit)
	m.operations.cancel(operationEditRecord)
}

func (m *model) cancelAllRequests() {
	m.operations.cancelAll()
}

type networkBusyState struct {
	operation operationKind
	message   string
	inline    bool
}

var networkOperationPriority = [...]operationKind{
	operationCreateRecord,
	operationEditRecord,
	operationDeleteRecord,
	operationCurrentUser,
	operationLogin,
	operationRegister,
	operationServerStatus,
	operationListRecords,
	operationViewRecord,
	operationLoadRecordForEdit,
}

var networkBusyStates = [operationCount]networkBusyState{
	operationCurrentUser:       {operation: operationCurrentUser, message: "Checking current user", inline: true},
	operationLogin:             {operation: operationLogin, message: "Logging in", inline: true},
	operationRegister:          {operation: operationRegister, message: "Registering", inline: true},
	operationServerStatus:      {operation: operationServerStatus, message: "Checking server", inline: true},
	operationListRecords:       {operation: operationListRecords, message: "Loading records", inline: true},
	operationViewRecord:        {operation: operationViewRecord, message: "Loading record", inline: true},
	operationLoadRecordForEdit: {operation: operationLoadRecordForEdit, message: "Loading record", inline: true},
	operationCreateRecord:      {operation: operationCreateRecord, message: "Creating record", inline: true},
	operationEditRecord:        {operation: operationEditRecord, message: "Saving record", inline: true},
	operationDeleteRecord:      {operation: operationDeleteRecord, message: "Deleting record", inline: true},
}

func (m model) currentNetworkBusyState() networkBusyState {
	for _, operation := range networkOperationPriority {
		if !m.operations.pending(operation) {
			continue
		}

		state := networkBusyStates[operation]

		if operation == operationCurrentUser && m.authentication.currentUserCheck == currentUserCheckRestore {
			state.message = "Restoring session"
			state.inline = false
		}

		return state
	}

	return networkBusyState{operation: operationNone}
}

func (m model) networkBusy() bool {
	return m.currentNetworkBusyState().operation != operationNone
}

func (m model) networkBusyPlacement() (windowPlacement, bool) {
	state := m.currentNetworkBusyState()
	if state.operation == operationNone || state.inline {
		return windowPlacement{}, false
	}

	return centeredWindowPlacement(
		renderNetworkBusyWindow(m.theme, networkBusyWindowWidth(m.width), state.message, m.spinnerFrameValue()),
		m.width,
		m.height,
	)
}

func networkBusyWindowWidth(screenWidth int) int {
	return clamp(screenWidth-34, 34, 46)
}

func renderNetworkBusyWindow(t theme, width int, message, spinnerFrame string) string {
	title := renderWindowTitle(t.windowTitle, width, "Please wait", spinnerFrame, true)
	bodyStyle := t.input
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#E4E4E4")).Background(lipgloss.Color("#0000AA"))
	body := bodyStyle.
		Width(width).
		Padding(2, 2).
		AlignHorizontal(lipgloss.Center).
		Render(messageStyle.Render(message + "..."))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func newActivitySpinner() spinner.Model {
	return spinner.New(spinner.WithSpinner(spinner.MiniDot))
}

func (m model) operationCommand(operation operationKind, command tea.Cmd) tea.Cmd {
	if command == nil || !operation.blocksInteraction() {
		return command
	}
	return tea.Batch(command, m.activitySpinner.Tick)
}

func (operation operationKind) blocksInteraction() bool {
	return operation > operationNone && operation < operationCount &&
		operation != operationLogout && operation != operationBinarySave
}

func (m model) spinnerPending() bool {
	return m.networkBusy()
}

func (m model) interactionBlocked() bool {
	return m.networkBusy()
}

func (m model) spinnerFrameValue() string {
	return m.activitySpinner.View()
}
