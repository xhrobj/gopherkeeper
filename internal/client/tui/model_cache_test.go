package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const cacheTestRecordID = "7a79b627-0473-48a0-a001-887e79419719"

func newCacheTestModel(t *testing.T, backend backendStub, authenticated bool) model {
	t.Helper()
	m := mustNewModel(
		t,
		context.Background(),
		config.Config{},
		"",
		buildinfo.Info{},
		staticBackendFactory(backend),
	)
	m.operations.cancel(operationCurrentUser)
	m.startupCmd = nil
	m.dialog = dialogNone
	m.authentication.session = authSession{state: authGuest}
	if authenticated {
		m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	}
	return m
}

func TestModel_CacheBrowseOpensForGuestAndKeepsPasswordOutOfModel(t *testing.T) {
	records := []recordmodel.RecordMetadata{{
		ID: cacheTestRecordID, Type: recordmodel.RecordTypeText, Title: "Offline note", Revision: 2,
	}}
	backend := backendStub{openCache: func(_ context.Context, login, password string) ([]recordmodel.RecordMetadata, error) {
		if login != "alice" || password != "old" {
			t.Fatalf("cache credentials = %q / %q", login, password)
		}
		return records, nil
	}}
	m := newCacheTestModel(t, backend, false)

	updated, command := m.activate(actionBrowseCache)
	m = updated.(model)
	if command != nil || m.dialog != dialogCacheBrowse || m.cacheFeature.form.focus != cacheBrowseLogin {
		t.Fatalf("open cache form = dialog %d focus %d command %t", m.dialog, m.cacheFeature.form.focus, command != nil)
	}

	m.cacheFeature.form.login.setValue("alice")
	m.cacheFeature.form.password.setValue("old")
	m.cacheFeature.form.focus = cacheBrowseSubmit
	updated, command = m.activateCacheBrowse()
	m = updated.(model)
	if command == nil || !m.operations.pending(operationOpenCache) {
		t.Fatalf("cache open pending = %t command %t", m.operations.pending(operationOpenCache), command != nil)
	}
	if m.cacheFeature.form.password.value != "" {
		t.Fatal("cache password remains in TUI model")
	}

	message := commandResult[cacheOpenResultMsg](t, command)
	updated, _ = m.handleCacheOpenResult(message)
	m = updated.(model)

	if m.dialog != dialogNone || !m.recordFeature.workspace.open ||
		m.recordFeature.workspace.source != recordSourceCache || len(m.recordFeature.workspace.records) != 1 {
		t.Fatalf("cache workspace = dialog %d workspace %#v", m.dialog, m.recordFeature.workspace)
	}
	plain := ansi.Strip(m.render())
	if !strings.Contains(plain, "Records from Cache: alice") || !strings.Contains(plain, "Offline note") {
		t.Fatalf("cache workspace not rendered: %q", plain)
	}
}

func TestModel_CacheBrowsePrefillsAuthenticatedLogin(t *testing.T) {
	m := newCacheTestModel(t, backendStub{}, true)
	updated, _ := m.activate(actionBrowseCache)
	m = updated.(model)

	if m.cacheFeature.form.login.value != "alice" || m.cacheFeature.form.focus != cacheBrowsePassword {
		t.Fatalf("cache form = login %q focus %d", m.cacheFeature.form.login.value, m.cacheFeature.form.focus)
	}
}

func TestModel_CacheBrowseFailureClearsPasswordAndReturnsToForm(t *testing.T) {
	m := newCacheTestModel(t, backendStub{openCache: func(context.Context, string, string) ([]recordmodel.RecordMetadata, error) {
		return nil, errors.New("failed to open encrypted local cache")
	}}, false)
	m.dialog = dialogCacheBrowse
	m.cacheFeature.form.login.setValue("alice")
	m.cacheFeature.form.password.setValue("wrong")
	m.cacheFeature.form.focus = cacheBrowseSubmit

	updated, command := m.activateCacheBrowse()
	m = updated.(model)
	message := commandResult[cacheOpenResultMsg](t, command)
	updated, _ = m.handleCacheOpenResult(message)
	m = updated.(model)

	if m.alert != alertError || m.alertReturnDialog != dialogCacheBrowse ||
		m.cacheFeature.form.password.value != "" || m.cacheFeature.form.focus != cacheBrowsePassword {
		t.Fatalf("cache failure state = alert %d return %d password %q focus %d",
			m.alert, m.alertReturnDialog, m.cacheFeature.form.password.value, m.cacheFeature.form.focus)
	}
}

func TestModel_CachedRecordViewUsesOpenCacheAndStaysReadOnly(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID: cacheTestRecordID, Type: recordmodel.RecordTypeCredentials, Title: "Mail", Revision: 4,
		},
		Payload: &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
	}
	backend := backendStub{getCached: func(_ context.Context, id string) (recordmodel.Record, error) {
		if id != cacheTestRecordID {
			t.Fatalf("cached record id = %q", id)
		}
		return record, nil
	}}
	m := newCacheTestModel(t, backend, false)
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
		records: []recordmodel.RecordMetadata{
			record.Metadata,
		},
	}

	updated, command := m.activate(actionViewCachedRecord)
	m = updated.(model)
	if command == nil || !m.operations.pending(operationViewCachedRecord) || m.dialog != dialogRecordView {
		t.Fatalf("cached view start = dialog %d pending %t", m.dialog, m.operations.pending(operationViewCachedRecord))
	}

	message := commandResult[cachedRecordViewResultMsg](t, command)
	updated, _ = m.handleCachedRecordViewResult(message)
	m = updated.(model)

	if m.recordFeature.view.status != recordViewReady || m.recordFeature.view.source != recordSourceCache {
		t.Fatalf("cached view = %#v", m.recordFeature.view)
	}
	plain := ansi.Strip(m.renderDialog())
	for _, part := range []string{"Record Credentials from Cache: alice", "Mail", "< Reveal >", "< Close >"} {
		if !strings.Contains(plain, part) {
			t.Fatalf("cached record view does not contain %q: %q", part, plain)
		}
	}

	definitions := m.currentMenuDefinitions()
	assertMenuActionDisabledState(t, definitions[menuRecord], actionEditRecord, true)
	assertMenuActionDisabledState(t, definitions[menuRecord], actionDeleteRecord, true)
}

func TestModel_CacheWorkspaceIgnoresDeleteAndClosesSession(t *testing.T) {
	closeCalls := 0
	m := newCacheTestModel(t, backendStub{closeCache: func() { closeCalls++ }}, false)
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		open:   true,
		state:  recordListReady,
		records: []recordmodel.RecordMetadata{{
			ID: cacheTestRecordID, Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1,
		}},
	}

	updated, _ := m.updateRecordWorkspace("delete")
	m = updated.(model)
	if m.dialog != dialogNone || len(m.recordFeature.workspace.records) != 1 {
		t.Fatal("Delete changed read-only cache workspace")
	}

	m.closeRecordWorkspace()
	if closeCalls != 1 || m.recordFeature.workspace.open {
		t.Fatalf("cache close = calls %d workspace %#v", closeCalls, m.recordFeature.workspace)
	}
}

func TestRenderCachedBinaryRecord_KeepsReadOnlySaveAction(t *testing.T) {
	state := recordViewState{
		source: recordSourceCache,
		login:  "alice",
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{
				ID: cacheTestRecordID, Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 2,
			},
			Payload: &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte("payload")},
		},
	}

	plain := ansi.Strip(renderRecordViewWindow(newTheme(), 72, 18, state, 1, false, false, ""))
	for _, part := range []string{"Record Binary from Cache: alice", "backup.bin", "< Save As... >", "< Close >"} {
		if !strings.Contains(plain, part) {
			t.Fatalf("cached binary view does not contain %q: %q", part, plain)
		}
	}
}

func TestModel_CacheMenuEnablesViewOnlyForCachedSelection(t *testing.T) {
	m := newCacheTestModel(t, backendStub{}, false)
	definitions := m.currentMenuDefinitions()
	assertMenuActionDisabledState(t, definitions[menuCache], actionBrowseCache, false)
	assertMenuActionDisabledState(t, definitions[menuCache], actionViewCachedRecord, true)

	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		open:   true,
		state:  recordListReady,
		records: []recordmodel.RecordMetadata{{
			ID: cacheTestRecordID, Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1,
		}},
	}
	definitions = m.currentMenuDefinitions()
	assertMenuActionDisabledState(t, definitions[menuCache], actionViewCachedRecord, false)
	assertMenuActionDisabledState(t, definitions[menuRecord], actionViewRecord, true)
}

func TestModel_SwitchingFromCacheToServerClosesCacheSession(t *testing.T) {
	closeCalls := 0
	m := newCacheTestModel(t, backendStub{closeCache: func() { closeCalls++ }}, true)
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		open:   true,
		state:  recordListReady,
	}

	command := m.beginOnlineRecordList()
	if command == nil || closeCalls != 1 {
		t.Fatalf("server browse = command %t cache close calls %d", command != nil, closeCalls)
	}
	if m.recordFeature.workspace.source != recordSourceServer || !m.operations.pending(operationListRecords) {
		t.Fatalf("server workspace = %#v pending %t", m.recordFeature.workspace, m.operations.pending(operationListRecords))
	}
}

func TestModel_StartingOnlineCreateClosesCacheWorkspace(t *testing.T) {
	closeCalls := 0
	m := newCacheTestModel(t, backendStub{closeCache: func() { closeCalls++ }}, true)
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		open:   true,
		state:  recordListReady,
	}

	updated, command := m.activate(actionNewRecord)
	m = updated.(model)
	if command != nil || m.dialog != dialogRecordType || closeCalls != 1 {
		t.Fatalf("new server record = dialog %d command %t cache close calls %d", m.dialog, command != nil, closeCalls)
	}
	if m.recordFeature.workspace.open || m.recordFeature.workspace.source != recordSourceNone {
		t.Fatalf("cache workspace remained open: %#v", m.recordFeature.workspace)
	}
}

func TestRenderCacheBrowseWindow_UsesPurpleBody(t *testing.T) {
	theme := newTheme()
	form := newCacheBrowseForm("m11")
	window := renderCacheBrowseWindow(theme, cacheBrowseWindowWidth(80), form, false, false, "")
	if !strings.Contains(window, theme.recordFormLabel.Render("Login")) {
		t.Fatal("cache browse form does not use the purple record-form label style")
	}
	if strings.Contains(window, theme.label.Render("Login")) {
		t.Fatal("cache browse form still uses the cyan dialog label style")
	}
}
