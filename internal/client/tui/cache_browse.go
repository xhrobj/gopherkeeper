package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type cacheBrowseFocus int

const (
	cacheBrowseLogin cacheBrowseFocus = iota
	cacheBrowsePassword
	cacheBrowseSubmit
	cacheBrowseCancel
	cacheBrowseFocusCount
)

type cacheBrowseForm struct {
	login    textField
	password textField
	focus    cacheBrowseFocus
}

func newCacheBrowseForm(login string) cacheBrowseForm {
	form := cacheBrowseForm{
		login:    newASCIITextField(strings.TrimSpace(login), false),
		password: newASCIITextField("", true),
		focus:    cacheBrowseLogin,
	}

	if form.login.value != "" {
		form.focus = cacheBrowsePassword
	}

	return form
}

func (form cacheBrowseForm) canSubmit() bool {
	return strings.TrimSpace(form.login.value) != "" && form.password.value != ""
}

func (form *cacheBrowseForm) move(step int, submitDisabled bool) {
	count := int(cacheBrowseFocusCount)

	for range count {
		form.focus = cacheBrowseFocus((int(form.focus) + step + count) % count)
		if !submitDisabled || form.focus != cacheBrowseSubmit {
			return
		}
	}
}

func (form *cacheBrowseForm) setFocus(focus cacheBrowseFocus, submitDisabled bool) {
	if focus < 0 || focus >= cacheBrowseFocusCount || submitDisabled && focus == cacheBrowseSubmit {
		return
	}

	form.focus = focus

	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *cacheBrowseForm) activeField() *textField {
	switch form.focus {
	case cacheBrowseLogin:
		return &form.login
	case cacheBrowsePassword:
		return &form.password
	default:
		return nil
	}
}

func (form *cacheBrowseForm) insert(value string) {
	if field := form.activeField(); field != nil {
		field.insert(value)
	}
}

func (form *cacheBrowseForm) insertKey(key string) bool {
	if field := form.activeField(); field != nil {
		return field.insertKey(key)
	}
	return false
}

func (form *cacheBrowseForm) backspace() {
	if field := form.activeField(); field != nil {
		field.backspace()
	}
}

func (form *cacheBrowseForm) delete() {
	if field := form.activeField(); field != nil {
		field.delete()
	}
}

func (form *cacheBrowseForm) moveCursor(step int) {
	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *cacheBrowseForm) moveCursorToStart() {
	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *cacheBrowseForm) moveCursorToEnd() {
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

type cacheOpenResultMsg struct {
	requestID uint64
	login     string
	records   []recordmodel.RecordMetadata
	err       error
}

func cacheOpenCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	login string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		records, err := backend.OpenCache(ctx, login, password)
		return cacheOpenResultMsg{requestID: requestID, login: login, records: records, err: err}
	}
}

func cleanCacheOpenError(err error) string {
	if err == nil {
		return "Unknown local cache error"
	}

	if errors.Is(err, context.Canceled) {
		return "Opening local cache canceled"
	}

	return cleanFailureMessage(err, "Unable to open local cache")
}

func (m model) updateCacheBrowse(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationOpenCache) {
		return m, nil
	}

	submitDisabled := !m.cacheFeature.form.canSubmit()

	switch key {
	case "esc":
		m.closeCacheBrowse()
	case "tab", "down":
		m.cacheFeature.form.move(1, submitDisabled)
	case "shift+tab", "up":
		m.cacheFeature.form.move(-1, submitDisabled)
	case "left":
		if m.cacheFeature.form.activeField() != nil {
			m.cacheFeature.form.moveCursor(-1)
		} else {
			m.cacheFeature.form.move(-1, submitDisabled)
		}
	case "right":
		if m.cacheFeature.form.activeField() != nil {
			m.cacheFeature.form.moveCursor(1)
		} else {
			m.cacheFeature.form.move(1, submitDisabled)
		}
	case "home":
		m.cacheFeature.form.moveCursorToStart()
	case "end":
		m.cacheFeature.form.moveCursorToEnd()
	case "backspace":
		m.cacheFeature.form.backspace()
	case "delete":
		m.cacheFeature.form.delete()
	case "enter":
		return m.activateCacheBrowse()
	default:
		m.cacheFeature.form.insertKey(key)
	}

	return m, nil
}

func (m model) activateCacheBrowse() (tea.Model, tea.Cmd) {
	switch m.cacheFeature.form.focus {
	case cacheBrowseSubmit:
		if !m.cacheFeature.form.canSubmit() || m.operations.pending(operationOpenCache) {
			return m, nil
		}

		login := strings.TrimSpace(m.cacheFeature.form.login.value)
		password := m.cacheFeature.form.password.value
		m.cacheFeature.form.password.setValue("")
		m.closeRecordWorkspace()

		requestCtx, requestID := m.operations.begin(m.ctx, operationOpenCache)
		return m, m.operationCommand(
			operationOpenCache,
			cacheOpenCommand(requestCtx, m.backend, requestID, login, password),
		)
	case cacheBrowseCancel:
		m.closeCacheBrowse()
	default:
		m.cacheFeature.form.move(1, !m.cacheFeature.form.canSubmit())
	}

	return m, nil
}

func (m model) handleCacheOpenResult(msg cacheOpenResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationOpenCache, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationOpenCache)
	m.cacheFeature.form.password.setValue("")

	if msg.err != nil {
		m.cacheFeature.form.focus = cacheBrowsePassword
		m.showAlert(alertError, "Unable to open cache", cleanCacheOpenError(msg.err), dialogCacheBrowse)
		return m, nil
	}

	m.recordFeature.workspace.beginCache(msg.login)
	m.recordFeature.workspace.apply(msg.records, recordWorkspacePageSize(m.height))
	m.cacheFeature.form = newCacheBrowseForm(msg.login)
	m.dialog = dialogNone
	m.activeButton = 0

	return m, nil
}

func (m *model) closeCacheBrowse() {
	m.operations.cancel(operationOpenCache)
	m.cacheFeature.form = newCacheBrowseForm(m.cacheFeature.form.login.value)
	m.dialog = dialogNone
	m.activeButton = 0
}

func (m *model) clearCacheState() {
	m.operations.cancel(operationOpenCache)
	m.cacheFeature.form = newCacheBrowseForm("")
}
