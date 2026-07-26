package tui

import (
	"context"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/binaryfile"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	minimumWidth  = 62
	minimumHeight = 23
	defaultWidth  = 80
	defaultHeight = 25
)

type dialogID int

const (
	dialogNone dialogID = iota
	dialogLogin
	dialogRegister
	dialogCurrentUser
	dialogControls
	dialogAbout
	dialogServerStatus
	dialogConfig
	dialogPathPicker
	dialogRecordView
	dialogBinarySave
	dialogRecordType
	dialogRecordCreate
	dialogRecordEdit
	dialogRecordDelete
	dialogCacheBrowse
	dialogSync
	dialogSyncResult
)

type alertState int

const (
	alertNone alertState = iota
	alertError
	alertNotice
)

type configSaver func(string, config.Config) error

type serverStatusState int

const (
	serverStatusIdle serverStatusState = iota
	serverStatusReady
	serverStatusFailed
)

type model struct {
	width             int
	height            int
	config            config.Config
	configFile        string
	saveConfig        configSaver
	info              buildinfo.Info
	theme             theme
	menuFocused       bool
	dropdownOpen      bool
	activeMenu        int
	selectedItem      int
	dialog            dialogID
	activeButton      int
	configForm        configForm
	pathPicker        pathPicker
	backend           Backend
	backendFactory    BackendFactory
	startupCmd        tea.Cmd
	authentication    authFeatureState
	recordFeature     recordFeatureState
	cacheFeature      cacheFeatureState
	syncFeature       syncFeatureState
	readBinaryFile    binaryFileReader
	writeBinaryFile   binaryFileWriter
	alert             alertState
	alertTitle        string
	alertMessage      string
	alertHighlight    string
	alertReturnDialog dialogID
	openURL           openURLFunc
	operationDone     <-chan struct{}
	statusState       serverStatusState
	statusValue       string
	statusFailure     serverStatusFailure
	operations        operationCoordinator
	statusMinDuration time.Duration
	activitySpinner   spinner.Model
}

func newModel(
	ctx context.Context,
	cfg config.Config,
	configFile string,
	info buildinfo.Info,
	backendFactory BackendFactory,
) (model, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	backend, err := createBackend(backendFactory, cfg)
	if err != nil {
		return model{}, err
	}

	m := model{
		width:          defaultWidth,
		height:         defaultHeight,
		config:         cfg,
		configFile:     configFile,
		saveConfig:     config.Save,
		info:           info,
		theme:          newTheme(),
		dialog:         dialogNone,
		activeButton:   0,
		backend:        backend,
		backendFactory: backendFactory,
		authentication: authFeatureState{
			session:      authSession{state: authUnknown},
			loginForm:    newLoginForm(),
			registerForm: newRegisterForm(),
		},
		recordFeature:     recordFeatureState{binarySaveForm: newBinarySaveForm("")},
		cacheFeature:      cacheFeatureState{form: newCacheBrowseForm("")},
		syncFeature:       syncFeatureState{form: newSyncForm()},
		configForm:        newConfigForm(cfg),
		readBinaryFile:    binaryfile.Read,
		writeBinaryFile:   binaryfile.Write,
		openURL:           openExternalURL,
		operationDone:     ctx.Done(),
		statusState:       serverStatusIdle,
		statusMinDuration: 500 * time.Millisecond,
		activitySpinner:   newActivitySpinner(),
	}

	m.startupCmd = m.beginCurrentUserCheck(currentUserCheckRestore)

	return m, nil
}

func (m model) Init() tea.Cmd {
	return m.startupCmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		return m.updateWindowSize(typed)
	case spinner.TickMsg:
		return m.updateSpinner(typed)
	case tea.MouseClickMsg:
		return m.updateMouse(typed)
	case tea.MouseWheelMsg:
		return m.updateMouseWheel(typed)
	case tea.PasteMsg:
		return m.updatePaste(typed)
	case tea.KeyPressMsg:
		return m.updateKeyPress(typed)
	}

	if updated, command, handled := m.updateResultMessage(msg); handled {
		return updated, command
	}

	if m.dialog == dialogPathPicker {
		return m.updatePathPicker(msg)
	}

	return m, nil
}

// authFeatureState объединяет пользовательское состояние сценариев авторизации.
type authFeatureState struct {
	session          authSession
	currentUserCheck currentUserCheckMode
	loginForm        loginForm
	registerForm     registerForm
}

// recordFeatureState объединяет состояние рабочего пространства и CRUD-сценариев записей.
type recordFeatureState struct {
	workspace      recordWorkspace
	view           recordViewState
	binarySaveForm binarySaveForm
	typePicker     recordTypePicker
	createForm     recordForm
	edit           recordEditState
	deletion       recordDeleteState
}

// cacheFeatureState хранит форму открытия локального кеша.
type cacheFeatureState struct {
	form cacheBrowseForm
}

// syncFeatureState объединяет состояние формы и результата синхронизации.
type syncFeatureState struct {
	form   syncForm
	result SyncSummary
}
