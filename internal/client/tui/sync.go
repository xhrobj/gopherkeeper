package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	domainmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type syncFocus int

const (
	syncPassword syncFocus = iota
	syncSubmit
	syncCancel
	syncFocusCount
)

type syncForm struct {
	password textField
	focus    syncFocus
}

func newSyncForm() syncForm {
	return syncForm{
		password: newASCIITextField("", true),
		focus:    syncPassword,
	}
}

func (form syncForm) canSubmit() bool {
	return form.password.value != ""
}

func (form *syncForm) move(step int, submitDisabled bool) {
	count := int(syncFocusCount)

	for range count {
		form.focus = syncFocus((int(form.focus) + step + count) % count)
		if !submitDisabled || form.focus != syncSubmit {
			return
		}
	}
}

func (form *syncForm) setFocus(focus syncFocus, submitDisabled bool) {
	if focus < 0 || focus >= syncFocusCount || submitDisabled && focus == syncSubmit {
		return
	}

	form.focus = focus
	if focus == syncPassword {
		form.password.moveCursorToEnd()
	}
}

func (form *syncForm) insert(value string) {
	if form.focus == syncPassword {
		form.password.insert(value)
	}
}

func (form *syncForm) insertKey(key string) bool {
	return form.focus == syncPassword && form.password.insertKey(key)
}

func (form *syncForm) backspace() {
	if form.focus == syncPassword {
		form.password.backspace()
	}
}

func (form *syncForm) delete() {
	if form.focus == syncPassword {
		form.password.delete()
	}
}

func (form *syncForm) moveCursor(step int) {
	if form.focus == syncPassword {
		form.password.moveCursor(step)
	}
}

func (form *syncForm) moveCursorToStart() {
	if form.focus == syncPassword {
		form.password.moveCursorToStart()
	}
}

func (form *syncForm) moveCursorToEnd() {
	if form.focus == syncPassword {
		form.password.moveCursorToEnd()
	}
}

type syncResultMsg struct {
	requestID uint64
	result    SyncSummary
	err       error
}

func syncCommand(ctx context.Context, backend Backend, requestID uint64, password string) tea.Cmd {
	return func() tea.Msg {
		result, err := backend.Sync(ctx, password)
		return syncResultMsg{requestID: requestID, result: result, err: err}
	}
}

func cleanSyncError(err error) string {
	if err == nil {
		return "Unknown synchronization error"
	}

	if errors.Is(err, context.Canceled) {
		return "Synchronization canceled"
	}

	if errors.Is(err, domainmodel.ErrInvalidCredentials) {
		return "Invalid login or password"
	}

	return cleanFailureMessage(err, "Unable to synchronize local cache")
}

func (m model) updateSync(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationSync) {
		return m, nil
	}

	submitDisabled := !m.syncFeature.form.canSubmit()
	switch key {
	case "esc":
		m.closeSync()
	case "tab", "down":
		m.syncFeature.form.move(1, submitDisabled)
	case "shift+tab", "up":
		m.syncFeature.form.move(-1, submitDisabled)
	case "left":
		if m.syncFeature.form.focus == syncPassword {
			m.syncFeature.form.moveCursor(-1)
		} else {
			m.syncFeature.form.move(-1, submitDisabled)
		}
	case "right":
		if m.syncFeature.form.focus == syncPassword {
			m.syncFeature.form.moveCursor(1)
		} else {
			m.syncFeature.form.move(1, submitDisabled)
		}
	case "home":
		m.syncFeature.form.moveCursorToStart()
	case "end":
		m.syncFeature.form.moveCursorToEnd()
	case "backspace":
		m.syncFeature.form.backspace()
	case "delete":
		m.syncFeature.form.delete()
	case "enter":
		return m.activateSync()
	default:
		m.syncFeature.form.insertKey(key)
	}

	return m, nil
}

func (m model) activateSync() (tea.Model, tea.Cmd) {
	switch m.syncFeature.form.focus {
	case syncSubmit:
		if !m.syncFeature.form.canSubmit() || m.operations.pending(operationSync) {
			return m, nil
		}
		password := m.syncFeature.form.password.value
		m.syncFeature.form.password.setValue("")
		requestCtx, requestID := m.operations.begin(m.ctx, operationSync)
		return m, m.operationCommand(operationSync, syncCommand(requestCtx, m.backend, requestID, password))
	case syncCancel:
		m.closeSync()
	default:
		m.syncFeature.form.move(1, !m.syncFeature.form.canSubmit())
	}

	return m, nil
}

func (m model) handleSyncResult(msg syncResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationSync, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationSync)
	m.syncFeature.form = newSyncForm()

	if msg.err != nil {
		if isNotLoggedIn(msg.err) {
			m.handleSessionExpired()
			return m, nil
		}

		m.showAlert(alertError, "Synchronization failed", cleanSyncError(msg.err), dialogSync)

		return m, nil
	}

	m.syncFeature.result = msg.result
	m.dialog = dialogSyncResult
	m.activeButton = 0

	if m.recordFeature.workspace.open && m.recordFeature.workspace.source == recordSourceServer {
		return m, m.beginOnlineRecordList()
	}

	return m, nil
}

func (m *model) closeSync() {
	m.operations.cancel(operationSync)
	m.syncFeature = syncFeatureState{form: newSyncForm()}
	m.dialog = dialogNone
	m.activeButton = 0
}

func (m *model) clearSyncState() {
	m.operations.cancel(operationSync)
	m.syncFeature = syncFeatureState{form: newSyncForm()}
}
