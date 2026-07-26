package tui

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func newRecordsTestModel(t *testing.T, cfg config.Config, backend recordsBackendStub) model {
	m := mustNewModel(t,
		context.Background(),
		cfg,
		"",
		buildinfo.Info{},
		staticBackendFactory(backend),
	)
	m.operations.cancel(operationCurrentUser)
	m.startupCmd = nil
	m.authentication.session = authSession{state: authGuest}
	m.dialog = dialogNone
	return m
}

func TestModel_BeginOnlineRecordList(t *testing.T) {
	backend := recordsBackendStub{
		listRecords: func(context.Context) ([]recordmodel.RecordMetadata, error) {
			return []recordmodel.RecordMetadata{{ID: "42"}}, nil
		},
	}
	m := newRecordsTestModel(t, config.Config{}, backend)

	if m.beginOnlineRecordList() != nil {
		t.Fatal("guest started online record loading")
	}

	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	command := m.beginOnlineRecordList()
	if command == nil || !m.operations.request(operationListRecords).pending {
		t.Fatal("online record loading did not start")
	}
	if !m.recordFeature.workspace.open || m.recordFeature.workspace.state != recordListLoading {
		t.Fatalf("workspace = %#v", m.recordFeature.workspace)
	}

	message := commandResult[recordListResultMsg](t, command)
	if message.requestID != m.operations.request(operationListRecords).id || len(message.records) != 1 {
		t.Fatalf("message = %#v", message)
	}
}

func TestModel_RecordListResultAppliesCurrentRequest(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.recordFeature.workspace.beginServer()
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42

	updated, _ := m.Update(recordListResultMsg{
		requestID: 42,
		records:   []recordmodel.RecordMetadata{{ID: "record-id", Title: "Server record"}},
	})
	got := updated.(model)

	if got.operations.request(operationListRecords).pending || got.recordFeature.workspace.state != recordListReady {
		t.Fatalf("result state = request %t workspace %d", got.operations.request(operationListRecords).pending, got.recordFeature.workspace.state)
	}
	if len(got.recordFeature.workspace.records) != 1 || got.recordFeature.workspace.records[0].Title != "Server record" {
		t.Fatalf("workspace = %#v", got.recordFeature.workspace)
	}
}

func TestModel_RecordListResultShowsSafeError(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.recordFeature.workspace.beginServer()
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42

	updated, _ := m.Update(recordListResultMsg{
		requestID: 42,
		err:       errors.New("list records: connection refused"),
	})
	got := updated.(model)

	if got.recordFeature.workspace.state != recordListFailed || got.alert != alertError {
		t.Fatalf("workspace state = %d, alert = %d", got.recordFeature.workspace.state, got.alert)
	}
	if got.alertMessage != "Connection refused" {
		t.Fatalf("alert message = %q", got.alertMessage)
	}
}

func TestModel_RecordListRefreshKeepsPreviousTableUntilResult(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{
		listRecords: func(context.Context) ([]recordmodel.RecordMetadata, error) {
			return []recordmodel.RecordMetadata{{ID: "new", Title: "New"}}, nil
		},
	})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		open:     true,
		state:    recordListReady,
		records:  []recordmodel.RecordMetadata{{ID: "old", Title: "Old"}},
		selected: 0,
	}

	command := m.beginOnlineRecordList()
	if command == nil || !m.operations.request(operationListRecords).pending {
		t.Fatal("refresh request did not start")
	}
	if m.recordFeature.workspace.state != recordListReady || len(m.recordFeature.workspace.records) != 1 || m.recordFeature.workspace.records[0].Title != "Old" {
		t.Fatalf("previous table was cleared while refreshing: %#v", m.recordFeature.workspace)
	}

	result := commandResult[recordListResultMsg](t, command)
	updated, _ := m.Update(result)
	got := updated.(model)
	if got.recordFeature.workspace.state != recordListReady || len(got.recordFeature.workspace.records) != 1 || got.recordFeature.workspace.records[0].Title != "New" {
		t.Fatalf("refreshed table = %#v", got.recordFeature.workspace)
	}
}

func TestModel_RecordListRefreshErrorKeepsPreviousTable(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.recordFeature.workspace = recordWorkspace{
		open:     true,
		state:    recordListReady,
		records:  []recordmodel.RecordMetadata{{ID: "old", Title: "Old"}},
		selected: 0,
	}
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42

	updated, _ := m.Update(recordListResultMsg{requestID: 42, err: errors.New("connection refused")})
	got := updated.(model)
	if got.recordFeature.workspace.state != recordListReady || len(got.recordFeature.workspace.records) != 1 || got.recordFeature.workspace.records[0].Title != "Old" {
		t.Fatalf("refresh error cleared previous table: %#v", got.recordFeature.workspace)
	}
	if got.alert != alertError {
		t.Fatalf("alert = %d, want error", got.alert)
	}
}

func TestModel_RecordListSessionExpiryResetsAuthenticatedUI(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogAbout
	m.activeButton = 1
	m.menuFocused = true
	m.dropdownOpen = true
	m.activeMenu = int(menuRecord)
	m.selectedItem = 3
	m.recordFeature.workspace = recordWorkspace{
		open:    true,
		state:   recordListReady,
		records: []recordmodel.RecordMetadata{{ID: "42", Title: "Old record"}},
	}
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42

	updated, _ := m.Update(recordListResultMsg{
		requestID: 42,
		err:       errors.Join(errors.New("list records"), usecase.ErrNotLoggedIn),
	})
	got := updated.(model)

	if got.authentication.session.state != authGuest || got.authentication.session.login != "" {
		t.Fatalf("auth after expiry = %#v", got.authentication.session)
	}
	if got.dialog != dialogNone || got.recordFeature.workspace.open || len(got.recordFeature.workspace.records) != 0 || got.recordFeature.workspace.failure != "" {
		t.Fatalf("windows after expiry: dialog = %d, records = %#v", got.dialog, got.recordFeature.workspace)
	}
	if got.menuFocused || got.dropdownOpen || got.activeMenu != int(menuSystem) || got.selectedItem != 0 {
		t.Fatalf("menu after expiry = focused %t dropdown %t active %d selected %d",
			got.menuFocused, got.dropdownOpen, got.activeMenu, got.selectedItem)
	}
	if got.alert != alertError || got.alertTitle != sessionExpiredTitle || got.alertMessage != sessionExpiredMessage {
		t.Fatalf("expiry alert = state %d title %q message %q", got.alert, got.alertTitle, got.alertMessage)
	}

	definitions := got.currentMenuDefinitions()
	account := definitions[menuAccount].items
	if account[0].disabled || account[1].disabled || !account[2].disabled || !account[4].disabled {
		t.Fatalf("Account menu is not in guest state: %#v", account)
	}
	if !definitions[menuRecord].disabled || !definitions[menuWindow].disabled {
		t.Fatalf("protected menus remain enabled: Records = %#v Window = %#v",
			definitions[menuRecord], definitions[menuWindow])
	}

	view := got.View().Content
	assertViewContains(t, view, sessionExpiredTitle, sessionExpiredMessage, "< OK >")
	assertViewExcludes(t, view, "Unable to load records", "Session expired, please login again", "Old record")
}

func TestModel_RecordListResultIgnoresStaleRequest(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.recordFeature.workspace.beginServer()
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 43

	updated, _ := m.Update(recordListResultMsg{
		requestID: 42,
		records:   []recordmodel.RecordMetadata{{ID: "stale"}},
	})
	got := updated.(model)

	if got.recordFeature.workspace.state != recordListLoading || len(got.recordFeature.workspace.records) != 0 || !got.operations.request(operationListRecords).pending {
		t.Fatalf("stale result changed model: %#v", got.recordFeature.workspace)
	}
}

func TestModel_RecordWorkspaceNavigationAndClose(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.height = 20
	m.recordFeature.workspace = recordWorkspace{
		open:    true,
		state:   recordListReady,
		records: make([]recordmodel.RecordMetadata, 12),
	}

	updated, _ := m.updateRecordWorkspace("pgdown")
	m = updated.(model)
	if m.recordFeature.workspace.selected == 0 {
		t.Fatal("PgDn did not move record selection")
	}
	updated, _ = m.updateRecordWorkspace("end")
	m = updated.(model)
	if m.recordFeature.workspace.selected != 11 {
		t.Fatalf("End selection = %d, want 11", m.recordFeature.workspace.selected)
	}
	updated, _ = m.updateRecordWorkspace("home")
	m = updated.(model)
	if m.recordFeature.workspace.selected != 0 {
		t.Fatalf("Home selection = %d, want 0", m.recordFeature.workspace.selected)
	}
	updated, _ = m.updateRecordWorkspace("esc")
	got := updated.(model)
	if got.recordFeature.workspace.open {
		t.Fatalf("closed workspace = %#v", got.recordFeature.workspace)
	}
}

func TestModel_RecordMenuAvailability(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	definitions := m.currentMenuDefinitions()
	recordsMenu := definitions[menuRecord]
	if !recordsMenu.disabled || !recordsMenu.items[0].disabled {
		t.Fatalf("guest Records menu = %#v", recordsMenu)
	}

	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	definitions = m.currentMenuDefinitions()
	recordsMenu = definitions[menuRecord]
	if recordsMenu.disabled || recordsMenu.items[0].disabled {
		t.Fatalf("authenticated Records menu = %#v", recordsMenu)
	}

	m.recordFeature.workspace.open = true
	definitions = m.currentMenuDefinitions()
	if definitions[menuWindow].disabled || definitions[menuWindow].items[0].disabled {
		t.Fatal("Window menu is disabled with records workspace open")
	}
	if action := m.currentAction(); action != actionBrowseRecords {
		t.Fatalf("current action = %d, want Browse", action)
	}
}

func TestModel_ActivateRecordActions(t *testing.T) {
	backend := recordsBackendStub{
		listRecords: func(context.Context) ([]recordmodel.RecordMetadata, error) { return nil, nil },
	}
	m := newRecordsTestModel(t, config.Config{}, backend)
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}

	updated, command := m.activate(actionBrowseRecords)
	m = updated.(model)
	if command == nil || !m.recordFeature.workspace.open {
		t.Fatalf("Browse state = command %t workspace %#v", command != nil, m.recordFeature.workspace)
	}

	m.dialog = dialogNone
	m.recordFeature.workspace.open = true
	updated, _ = m.activate(actionCloseWindow)
	got := updated.(model)
	if got.recordFeature.workspace.open {
		t.Fatalf("Close Active Window left workspace open: %#v", got.recordFeature.workspace)
	}
}

func TestModel_CurrentUserResultStartsRecordLoading(t *testing.T) {
	backend := recordsBackendStub{
		listRecords: func(context.Context) ([]recordmodel.RecordMetadata, error) { return nil, nil },
	}
	m := newRecordsTestModel(t, config.Config{}, backend)
	m.authentication.session = authSession{state: authUnknown}
	m.operations.request(operationCurrentUser).pending = true
	m.operations.request(operationCurrentUser).id = 42

	updated, command := m.Update(currentUserResultMsg{requestID: 42, login: "alice"})
	got := updated.(model)
	if command == nil || !got.authentication.session.authenticated() || !got.recordFeature.workspace.open || !got.operations.request(operationListRecords).pending {
		t.Fatalf("session result state = command %t auth %#v records %#v", command != nil, got.authentication.session, got.recordFeature.workspace)
	}
}

func TestModel_ConfigChangeClosesRecordWorkspace(t *testing.T) {
	backend := recordsBackendStub{}
	m := newRecordsTestModel(t, config.Config{Address: "old.example:8443", CACertFile: "/tmp/old-ca.pem"}, backend)
	m.dialog = dialogConfig
	m.configForm = newConfigForm(config.Config{Address: "new.example:9443", CACertFile: "/tmp/new-ca.pem"})
	m.configForm.focus = configSave
	m.recordFeature.workspace = recordWorkspace{open: true, state: recordListReady}
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{Payload: &recordmodel.CredentialsPayload{Password: "secret"}},
	}
	m.operations.request(operationViewRecord).pending = true
	m.operations.request(operationViewRecord).id = 42
	m.backendFactory = staticBackendFactory(backend)

	updated, _ := m.activateConfig()
	got := updated.(model)

	if got.recordFeature.workspace.open {
		t.Fatalf("workspace survived config change: %#v", got.recordFeature.workspace)
	}
	if got.recordFeature.view.status != recordViewIdle || got.operations.request(operationViewRecord).pending {
		t.Fatalf("record view survived config change: state %#v request %#v", got.recordFeature.view, got.operations.request(operationViewRecord))
	}
}

func TestModel_LeavingRecordViewCancelsRequestAndIgnoresResult(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogRecordView
	m.recordFeature.view.begin(recordmodel.RecordMetadata{})
	m.operations.request(operationViewRecord).pending = true
	m.operations.request(operationViewRecord).id = 42
	canceled := false
	m.operations.request(operationViewRecord).cancel = func() { canceled = true }

	updated, _ := m.activate(actionAbout)
	got := updated.(model)
	if !canceled || got.operations.request(operationViewRecord).pending || got.recordFeature.view.status != recordViewIdle {
		t.Fatalf("record view was not left cleanly: canceled %t state %#v request %#v", canceled, got.recordFeature.view, got.operations.request(operationViewRecord))
	}
	if got.dialog != dialogAbout {
		t.Fatalf("dialog = %v, want About", got.dialog)
	}

	updated, _ = got.Update(recordViewResultMsg{
		requestID: 42,
		record:    recordmodel.Record{Payload: &recordmodel.CredentialsPayload{Password: "secret"}},
	})
	got = updated.(model)
	if got.dialog != dialogAbout || got.recordFeature.view.status != recordViewIdle || got.recordFeature.view.record.Payload != nil {
		t.Fatalf("stale record result changed model: dialog %v state %#v", got.dialog, got.recordFeature.view)
	}
}

func TestModel_LogoutLeavesRecordViewBeforeRequestCompletes(t *testing.T) {
	backend := recordsBackendStub{backendStub: backendStub{logout: func(context.Context) error { return nil }}}
	m := newRecordsTestModel(t, config.Config{}, backend)
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogRecordView
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{Payload: &recordmodel.CredentialsPayload{Password: "secret"}},
	}
	m.operations.request(operationViewRecord).pending = true
	m.operations.request(operationViewRecord).id = 42

	updated, command := m.activate(actionLogout)
	got := updated.(model)
	if command == nil || got.dialog != dialogNone || !got.operations.request(operationLogout).pending {
		t.Fatalf("logout state = command %t dialog %v pending %t", command != nil, got.dialog, got.operations.request(operationLogout).pending)
	}
	if got.recordFeature.view.status != recordViewIdle || got.recordFeature.view.record.Payload != nil || got.operations.request(operationViewRecord).pending {
		t.Fatalf("record view survived logout start: state %#v request %#v", got.recordFeature.view, got.operations.request(operationViewRecord))
	}

	updated, _ = got.Update(recordViewResultMsg{
		requestID: 42,
		record:    recordmodel.Record{Payload: &recordmodel.CredentialsPayload{Password: "secret"}},
	})
	got = updated.(model)
	if got.recordFeature.view.status != recordViewIdle || got.recordFeature.view.record.Payload != nil {
		t.Fatalf("stale record result was applied after logout: %#v", got.recordFeature.view)
	}
}

func TestModel_RecordViewResultUsesRecordDefaultButton(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogRecordView
	m.recordFeature.view.begin(recordmodel.RecordMetadata{})
	m.operations.request(operationViewRecord).pending = true
	m.operations.request(operationViewRecord).id = 42

	updated, _ := m.Update(recordViewResultMsg{
		requestID: 42,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin"},
		},
	})
	got := updated.(model)
	if got.recordFeature.view.status != recordViewReady || got.activeButton != recordViewDefaultButton(got.recordFeature.view.record) || got.activeButton != 1 {
		t.Fatalf("record result state = %#v active button = %d", got.recordFeature.view, got.activeButton)
	}
}

func TestModel_CancelAllRequestsCancelsRecordLoading(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42
	canceled := false
	m.operations.request(operationListRecords).cancel = func() { canceled = true }

	m.cancelAllRequests()

	if !canceled || m.operations.request(operationListRecords).pending || m.operations.request(operationListRecords).id != 43 {
		t.Fatalf("record request state = canceled %t pending %t id %d", canceled, m.operations.request(operationListRecords).pending, m.operations.request(operationListRecords).id)
	}
}

func TestModel_RecordsViewMenuOpensSelectedRecord(t *testing.T) {
	backend := recordsBackendStub{
		getRecord: func(_ context.Context, recordID string) (recordmodel.Record, error) {
			if recordID != "42" {
				t.Fatalf("record id = %q, want 42", recordID)
			}
			return recordmodel.Record{
				Metadata: recordmodel.RecordMetadata{ID: "42", Type: recordmodel.RecordTypeText, Title: "Note"},
				Payload:  &recordmodel.TextPayload{Text: "secret"},
			}, nil
		},
	}
	m := newRecordsTestModel(t, config.Config{}, backend)
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{{ID: "42", Type: recordmodel.RecordTypeText, Title: "Note"}}, 10)

	definitions := m.currentMenuDefinitions()
	viewDisabled := true
	for _, item := range definitions[menuRecord].items {
		if item.action == actionViewRecord {
			viewDisabled = item.disabled
			break
		}
	}
	if viewDisabled {
		t.Fatal("Records → View is disabled for a selected record")
	}

	updated, command := m.activate(actionViewRecord)
	m = updated.(model)
	if command == nil || m.dialog != dialogRecordView || !m.operations.request(operationViewRecord).pending {
		t.Fatalf("view state = dialog %d request %#v command %t", m.dialog, m.operations.request(operationViewRecord), command != nil)
	}
	updated, _ = m.Update(commandResult[recordViewResultMsg](t, command))
	got := updated.(model)
	if got.recordFeature.view.status != recordViewReady || got.recordFeature.view.record.Metadata.ID != "42" {
		t.Fatalf("record view = %#v", got.recordFeature.view)
	}
}
