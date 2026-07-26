package tui

import (
	"testing"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestPrepareDialogChange_CloseWindowKeepsDialogState(t *testing.T) {
	m := model{dialog: dialogLogin}
	m.authentication.loginForm = newLoginForm()
	m.authentication.loginForm.login.setValue("alice")

	m.prepareDialogChange(actionCloseWindow)

	if m.dialog != dialogLogin || m.authentication.loginForm.login.value != "alice" {
		t.Fatalf("close-window preparation changed login dialog: dialog=%d login=%q", m.dialog, m.authentication.loginForm.login.value)
	}
}

func TestPrepareDialogChange_ClearsLoginDialog(t *testing.T) {
	m := model{dialog: dialogLogin}
	m.authentication.loginForm = newLoginForm()
	m.authentication.loginForm.login.setValue("alice")
	_, _ = m.operations.begin(nil, operationLogin)

	m.prepareDialogChange(actionConfig)

	if m.authentication.loginForm.login.value != "" || m.operations.pending(operationLogin) {
		t.Fatalf("login state was not cleared: login=%q pending=%v", m.authentication.loginForm.login.value, m.operations.pending(operationLogin))
	}
}

func TestPrepareDialogChange_ClearsRegisterDialog(t *testing.T) {
	m := model{dialog: dialogRegister}
	m.authentication.registerForm = newRegisterForm()
	m.authentication.registerForm.login.setValue("alice")
	_, _ = m.operations.begin(nil, operationRegister)

	m.prepareDialogChange(actionAbout)

	if m.authentication.registerForm.login.value != "" || m.operations.pending(operationRegister) {
		t.Fatalf("register state was not cleared: login=%q pending=%v", m.authentication.registerForm.login.value, m.operations.pending(operationRegister))
	}
}

func TestPrepareDialogChange_CancelsServerStatusCheck(t *testing.T) {
	m := model{dialog: dialogServerStatus}
	_, _ = m.operations.begin(nil, operationServerStatus)

	m.prepareDialogChange(actionConfig)

	if m.operations.pending(operationServerStatus) {
		t.Fatal("server-status request remains pending")
	}
}

func TestPrepareDialogChange_ClearsConfigPathPicker(t *testing.T) {
	m := model{
		dialog: dialogPathPicker,
		pathPicker: pathPicker{
			target:           pathPickerCACert,
			rootDirectory:    "/workspace",
			currentDirectory: "/workspace/certs",
		},
	}

	m.prepareDialogChange(actionAbout)

	if m.pathPicker.rootDirectory != "" || m.pathPicker.currentDirectory != "" || len(m.pathPicker.entries) != 0 {
		t.Fatalf("config path picker was not cleared: %#v", m.pathPicker)
	}
}

func TestPrepareDialogChange_KeepsMatchingBinaryCreatePathPicker(t *testing.T) {
	m := model{
		dialog: dialogPathPicker,
		pathPicker: pathPicker{
			target:           pathPickerBinaryCreateFile,
			rootDirectory:    "/workspace",
			currentDirectory: "/workspace/files",
		},
	}

	m.prepareDialogChange(actionNewRecord)

	if m.pathPicker.target != pathPickerBinaryCreateFile || m.pathPicker.currentDirectory == "" {
		t.Fatalf("matching binary-create picker was cleared: %#v", m.pathPicker)
	}
}

func TestPrepareDialogChange_ClearsRecordViewFromBinarySavePicker(t *testing.T) {
	m := model{
		dialog: dialogPathPicker,
		pathPicker: pathPicker{
			target: pathPickerBinarySaveDirectory,
		},
	}
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
			Payload:  &recordmodel.BinaryPayload{Filename: "secret.bin", Data: []byte("secret")},
		},
	}
	m.recordFeature.binarySaveForm = newBinarySaveForm("secret.bin")
	_, _ = m.operations.begin(nil, operationViewRecord)
	_, _ = m.operations.begin(nil, operationBinarySave)

	m.prepareDialogChange(actionConfig)

	if m.recordFeature.view.status != recordViewIdle || m.recordFeature.binarySaveForm.fileName != "" {
		t.Fatalf("record view was not cleared: view=%#v save=%#v", m.recordFeature.view, m.recordFeature.binarySaveForm)
	}
	if m.operations.pending(operationViewRecord) || m.operations.pending(operationBinarySave) {
		t.Fatal("record-view requests remain pending")
	}
}

func TestPrepareDialogChange_ClearsRecordCreateState(t *testing.T) {
	m := model{dialog: dialogRecordCreate}
	m.recordFeature.typePicker.selected = 2
	m.recordFeature.createForm = newRecordCreateForm(recordmodel.RecordTypeCredentials)
	m.recordFeature.createForm.title.setValue("Account")
	_, _ = m.operations.begin(nil, operationCreateRecord)

	m.prepareDialogChange(actionConfig)

	if m.dialog != dialogNone || m.recordFeature.createForm.recordType != "" || m.recordFeature.typePicker.selected != 0 {
		t.Fatalf("record-create state was not cleared: dialog=%d form=%#v picker=%#v", m.dialog, m.recordFeature.createForm, m.recordFeature.typePicker)
	}
	if m.operations.pending(operationCreateRecord) {
		t.Fatal("create-record request remains pending")
	}
}

func TestPrepareDialogChange_ClearsRecordEditState(t *testing.T) {
	m := model{dialog: dialogRecordEdit}
	m.recordFeature.edit = recordEditState{
		status: recordEditReady,
		record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{ID: "record-id"}},
	}
	_, _ = m.operations.begin(nil, operationLoadRecordForEdit)
	_, _ = m.operations.begin(nil, operationEditRecord)

	m.prepareDialogChange(actionAbout)

	if m.dialog != dialogNone || m.recordFeature.edit.status != 0 || m.recordFeature.edit.record.Metadata.ID != "" {
		t.Fatalf("record-edit state was not cleared: dialog=%d edit=%#v", m.dialog, m.recordFeature.edit)
	}
	if m.operations.pending(operationLoadRecordForEdit) || m.operations.pending(operationEditRecord) {
		t.Fatal("record-edit requests remain pending")
	}
}

func TestPrepareDialogChange_ClearsRecordDeleteState(t *testing.T) {
	m := model{dialog: dialogRecordDelete}
	m.recordFeature.deletion = recordDeleteState{metadata: recordmodel.RecordMetadata{ID: "record-id"}}
	_, _ = m.operations.begin(nil, operationDeleteRecord)

	m.prepareDialogChange(actionAbout)

	if m.dialog != dialogNone || m.recordFeature.deletion.metadata.ID != "" || m.operations.pending(operationDeleteRecord) {
		t.Fatalf("record-delete state was not cleared: dialog=%d delete=%#v pending=%v", m.dialog, m.recordFeature.deletion, m.operations.pending(operationDeleteRecord))
	}
}

func TestPrepareDialogChange_ClearsCacheBrowseState(t *testing.T) {
	m := model{dialog: dialogCacheBrowse}
	m.cacheFeature.form = newCacheBrowseForm("alice")
	m.cacheFeature.form.password.setValue("secret")
	_, _ = m.operations.begin(nil, operationOpenCache)

	m.prepareDialogChange(actionAbout)

	if m.dialog != dialogNone || m.cacheFeature.form.login.value != "alice" || m.cacheFeature.form.password.value != "" {
		t.Fatalf("cache form was not reset: dialog=%d form=%#v", m.dialog, m.cacheFeature.form)
	}
	if m.operations.pending(operationOpenCache) {
		t.Fatal("open-cache request remains pending")
	}
}

func TestPrepareDialogChange_ClearsSyncState(t *testing.T) {
	m := model{dialog: dialogSyncResult}
	m.syncFeature.form = newSyncForm()
	m.syncFeature.form.password.setValue("secret")
	m.syncFeature.result = SyncSummary{Added: 2}
	_, _ = m.operations.begin(nil, operationSync)

	m.prepareDialogChange(actionAbout)

	if m.dialog != dialogNone || m.syncFeature.form.password.value != "" || m.syncFeature.result.Added != 0 {
		t.Fatalf("sync state was not cleared: dialog=%d state=%#v", m.dialog, m.syncFeature)
	}
	if m.operations.pending(operationSync) {
		t.Fatal("sync request remains pending")
	}
}

func TestPrepareDialogChange_KeepsStateForMatchingAction(t *testing.T) {
	tests := []struct {
		name      string
		dialog    dialogID
		action    actionID
		target    pathPickerTarget
		operation operationKind
	}{
		{name: "login", dialog: dialogLogin, action: actionLogin, operation: operationLogin},
		{name: "register", dialog: dialogRegister, action: actionRegister, operation: operationRegister},
		{name: "server status", dialog: dialogServerStatus, action: actionServerStatus, operation: operationServerStatus},
		{name: "config picker", dialog: dialogPathPicker, action: actionConfig, target: pathPickerCACert},
		{name: "binary save picker", dialog: dialogPathPicker, action: actionBrowseRecords, target: pathPickerBinarySaveDirectory},
		{name: "binary create picker", dialog: dialogPathPicker, action: actionNewRecord, target: pathPickerBinaryCreateFile},
		{name: "binary edit picker", dialog: dialogPathPicker, action: actionEditRecord, target: pathPickerBinaryEditFile},
		{name: "record view", dialog: dialogRecordView, action: actionEditRecord},
		{name: "record create", dialog: dialogRecordCreate, action: actionNewRecord},
		{name: "record edit", dialog: dialogRecordEdit, action: actionEditRecord},
		{name: "record delete", dialog: dialogRecordDelete, action: actionDeleteRecord},
		{name: "cache", dialog: dialogCacheBrowse, action: actionBrowseCache},
		{name: "sync", dialog: dialogSync, action: actionSynchronize},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := model{dialog: test.dialog}
			m.pathPicker.target = test.target
			if test.operation != operationNone {
				_, _ = m.operations.begin(nil, test.operation)
			}

			m.prepareDialogChange(test.action)

			if m.dialog != test.dialog || m.pathPicker.target != test.target {
				t.Fatalf("matching action changed state: dialog=%d target=%d", m.dialog, m.pathPicker.target)
			}
			if test.operation != operationNone && !m.operations.pending(test.operation) {
				t.Fatalf("matching action canceled operation %d", test.operation)
			}
			if test.operation != operationNone {
				m.operations.cancel(test.operation)
			}
		})
	}
}

func TestPrepareDialogChange_ClearsBinaryEditPathPicker(t *testing.T) {
	m := model{
		dialog: dialogPathPicker,
		pathPicker: pathPicker{
			target:           pathPickerBinaryEditFile,
			currentDirectory: "/workspace/files",
		},
	}
	m.recordFeature.edit = recordEditState{
		status: recordEditReady,
		record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{ID: "record-id"}},
	}

	m.prepareDialogChange(actionConfig)

	if m.dialog != dialogNone || m.pathPicker.currentDirectory != "" || m.recordFeature.edit.status != 0 {
		t.Fatalf("binary-edit picker was not cleared: dialog=%d picker=%#v edit=%#v", m.dialog, m.pathPicker, m.recordFeature.edit)
	}
}
