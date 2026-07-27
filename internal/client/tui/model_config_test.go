package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestModel_ConfigChangeRechecksSessionForNewRuntime(t *testing.T) {
	initial := config.Config{Address: "old.example:8443", CACertFile: "/tmp/ca.pem", SessionDir: "/tmp/old-session"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configAddress].setValue("new.example:9443")
	m.configForm.fields[configSessionDir].setValue("/tmp/new-session")
	m.configForm.focus = configSave

	var configured config.Config
	currentUserCalled := false
	m.backendFactory = func(cfg config.Config) (Backend, error) {
		configured = cfg
		return backendStub{
			currentUser: func(context.Context) (string, error) {
				currentUserCalled = true
				return "bob", nil
			},
		}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("config change did not start a session recheck")
	}
	if m.authentication.session.state != authUnknown || m.authentication.session.login != "" || m.dialog != dialogNone {
		t.Fatalf("state before recheck = auth %#v dialog %d", m.authentication.session, m.dialog)
	}

	updated, _ = m.Update(commandResult[currentUserResultMsg](t, cmd))
	got := updated.(model)
	if configured != got.config {
		t.Fatalf("configured runtime = %#v, want %#v", configured, got.config)
	}
	if !currentUserCalled {
		t.Fatal("CurrentUser was not called after runtime reconfiguration")
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" {
		t.Fatalf("state after recheck = %#v, want authenticated bob", got.authentication.session)
	}
}

func TestModel_ConfigChangeKeepsCurrentRuntimeWhenBackendCreationFails(t *testing.T) {
	initial := config.Config{Address: "old.example:8443", CACertFile: "/tmp/old-ca.pem"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configAddress].setValue("new.example:9443")
	m.configForm.fields[configCACertFile].setValue("/tmp/new-ca.pem")
	m.configForm.focus = configSave
	currentBackend := &backendStub{}
	m.backend = currentBackend
	want := errors.New("backend creation failed")
	m.backendFactory = func(config.Config) (Backend, error) {
		return nil, want
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)

	if cmd != nil {
		t.Fatal("failed runtime change started a command")
	}
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want config", got.dialog)
	}
	if got.config != initial {
		t.Fatalf("config = %#v, want %#v", got.config, initial)
	}
	if got.backend != currentBackend {
		t.Fatal("backend changed after factory error")
	}
	const wantMessage = "Backend creation failed"
	if got.configForm.errorMessage != wantMessage {
		t.Fatalf("config error = %q, want %q", got.configForm.errorMessage, wantMessage)
	}
}

func TestRenderConfigInputs_RespectCellWidthForWideUnicode(t *testing.T) {
	const width = 10
	theme := newTheme()

	values := []string{
		renderConfigReadOnlyInput(theme, theme.readOnly, "путь/界界界界界界", width, true),
		renderConfigReadOnlyInput(theme, theme.readOnly, "界界界界界界/путь", width, false),
		renderTextField(theme, newUnicodeTextField("путь/界界界界界界"), width, false),
		renderTextField(theme, newUnicodeTextField("путь/界界界界界界"), width, true),
	}

	for index, value := range values {
		if got := lipgloss.Width(ansi.Strip(value)); got != width {
			t.Fatalf("rendered config input %d width = %d, want %d", index, got, width)
		}
	}
}
func TestRenderConfigInput_ShowsCursorOnlyForFocusedField(t *testing.T) {
	theme := newTheme()
	field := newUnicodeTextField("abcd")
	field.cursor = 1

	focused := renderTextField(theme, field, 10, true)
	blurred := renderTextField(theme, field, 10, false)
	cursor := theme.inputCursor.Render("b")

	if !strings.Contains(focused, cursor) {
		t.Fatal("focused config input does not render the cursor")
	}
	if strings.Contains(blurred, cursor) {
		t.Fatal("blurred config input renders the cursor")
	}
}
func TestModel_SystemConfigMnemonicOpensPrefilledDialog(t *testing.T) {
	cfg := config.Config{
		Transport:   config.TransportGRPC,
		Address:     "server.example:8443",
		GRPCAddress: "server.example:9443",
		CACertFile:  "/tmp/ca.pem",
		SessionDir:  "/tmp/session",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, cfg, buildinfo.Info{})
	m.menuFocused = true

	updated, _ := m.Update(keyPress("s"))
	m = updated.(model)
	updated, _ = m.Update(keyPress("c"))
	got := updated.(model)

	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want config", got.dialog)
	}
	if got.configForm.config() != cfg {
		t.Fatalf("config form = %#v, want %#v", got.configForm.config(), cfg)
	}
	assertViewContains(t, got.View().Content,
		"Config",
		"Transport",
		"[X] gRPC",
		"HTTPS address",
		"server.example:8443",
		"gRPC address",
		"server.example:9443",
		"CA cert file",
		"/tmp/ca.pem",
		"Session dir",
		"/tmp/session",
		"Cache dir",
		"/tmp/cache",
		"< Save >",
		"< Cancel >",
	)
}
func TestModel_ConfigAcceptsTerminalPaste(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.fields[configAddress].setValue("")
	m.configForm.fields[configAddress].cursor = 0
	m.configForm.focus = configAddress

	updated, _ := m.Update(tea.PasteMsg{Content: "vault.example:9443\nignored.example:8080"})
	got := updated.(model)

	if got.configForm.fields[configAddress].value != "vault.example:9443" {
		t.Fatalf("pasted address = %q, want vault.example:9443", got.configForm.fields[configAddress].value)
	}
}
func TestModel_ConfigSaveUpdatesRuntimeConfig(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8080"}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.fields[configAddress].setValue("vault.example:9443")
	m.configForm.fields[configCACertFile].setValue("/etc/gopherkeeper/ca.pem")
	m.configForm.focus = configSave

	updated, _ := m.Update(keyPress("enter"))
	got := updated.(model)

	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
	if got.config.Address != "vault.example:9443" {
		t.Fatalf("address = %q, want vault.example:9443", got.config.Address)
	}
	if got.config.CACertFile != "/etc/gopherkeeper/ca.pem" {
		t.Fatalf("CA cert = %q", got.config.CACertFile)
	}
}
func TestModel_ConfigCancelKeepsResolvedConfig(t *testing.T) {
	initial := config.Config{Address: "localhost:8080", CacheDir: "/old/cache"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.fields[configCacheDir].setValue("/new/cache")
	m.configForm.focus = configCancel

	updated, _ := m.Update(keyPress("enter"))
	got := updated.(model)

	if got.config != initial {
		t.Fatalf("config = %#v, want %#v", got.config, initial)
	}
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
}
func TestRenderConfigWindow_ShowsMissingConfigPersistenceHint(t *testing.T) {
	view := renderConfigWindow(newTheme(), 82, newConfigForm(config.Config{}), "")
	plain := ansi.Strip(view)
	if !strings.Contains(plain, configFileNotPassed) {
		t.Fatalf("config view does not contain missing-config marker %q: %q", configFileNotPassed, plain)
	}
	if !strings.Contains(plain, "use --config for persistence") {
		t.Fatalf("config view does not explain persistence: %q", plain)
	}
}
func TestRenderConfigWindow_ShowsOnlyTwoYellowRequiredMarkers(t *testing.T) {
	theme := newTheme()
	view := renderConfigWindow(theme, 82, newConfigForm(config.Config{}), "")
	plain := ansi.Strip(view)

	if strings.Contains(plain, "* required") {
		t.Fatalf("config view still contains required legend: %q", plain)
	}
	if count := strings.Count(plain, "*"); count != 2 {
		t.Fatalf("required marker count = %d, want 2: %q", count, plain)
	}
	if count := strings.Count(view, theme.configRequiredYellow.Render("*")); count != 2 {
		t.Fatalf("yellow required marker count = %d, want 2", count)
	}
	for _, line := range strings.Split(plain, "\n") {
		if strings.Contains(line, "Session dir") && strings.Contains(line, "*") {
			t.Fatalf("Session dir still has a required marker: %q", line)
		}
	}
	if !strings.Contains(view, theme.configMissing.Render(configFileNotPassed)) {
		t.Fatal("missing config-file hint is not rendered with high-contrast style")
	}
}

func TestNewModel_CreatesBackendForInitialRuntime(t *testing.T) {
	cfg := config.Config{
		Address:    "vault.example:8443",
		CACertFile: "/tmp/ca.pem",
		SessionDir: "/tmp/session",
		CacheDir:   "/tmp/cache",
	}

	var configured config.Config
	m := mustNewModel(t,
		context.Background(),
		cfg,
		"",
		buildinfo.Info{},
		func(got config.Config) (Backend, error) {
			configured = got
			return backendStub{
				currentUser: func(context.Context) (string, error) {
					return "", nil
				},
			}, nil
		},
	)
	m.cancelAllRequests()

	if configured != cfg {
		t.Fatalf("configured runtime = %#v, want %#v", configured, cfg)
	}
}

func TestConfigForm_SaveRequiresActiveTransportAddress(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want bool
	}{
		{name: "all fields filled", cfg: config.Config{Address: "localhost:8888", CACertFile: "/tmp/ca.pem"}, want: true},
		{name: "only address filled", cfg: config.Config{Address: "localhost:8888"}, want: true},
		{name: "address empty", cfg: config.Config{CACertFile: "/tmp/ca.pem"}},
		{name: "address contains only spaces", cfg: config.Config{Address: "  ", CACertFile: "/tmp/ca.pem"}},
		{name: "gRPC address filled", cfg: config.Config{Transport: config.TransportGRPC, GRPCAddress: "localhost:9090"}, want: true},
		{name: "gRPC address empty", cfg: config.Config{Transport: config.TransportGRPC, Address: "localhost:8080"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := newConfigForm(test.cfg).canSave(); got != test.want {
				t.Fatalf("canSave() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestConfigForm_MoveSkipsDisabledSave(t *testing.T) {
	form := newConfigForm(config.Config{})
	form.focus = configCacheBrowse

	form.move(1)

	if form.focus != configCancel {
		t.Fatalf("focus = %d, want Cancel", form.focus)
	}
}

func TestRenderConfigWindow_DisablesSaveWithoutAddress(t *testing.T) {
	theme := newTheme()
	view := renderConfigWindow(theme, 82, newConfigForm(config.Config{CACertFile: "/tmp/ca.pem"}), "")

	if !strings.Contains(view, theme.buttonDisabled.Render("< Save >")) {
		t.Fatal("Config does not render disabled Save without server address")
	}
	if strings.Contains(ansi.Strip(view), "Server address is required") {
		t.Fatal("Config still renders the old required-field validation message")
	}
}

func TestRenderConfigWindow_EnablesSaveWithoutCACertificate(t *testing.T) {
	theme := newTheme()
	form := newConfigForm(config.Config{Address: "localhost:8888"})
	form.setFocus(configSave)
	view := renderConfigWindow(theme, 82, form, "")

	if !strings.Contains(view, theme.buttonActive.Render("< Save >")) {
		t.Fatal("Config does not enable Save when only the server address is filled")
	}
	if count := strings.Count(view, theme.configRequiredYellow.Render("*")); count != 2 {
		t.Fatalf("yellow required marker count = %d, want 2", count)
	}
}

func TestModel_ConfigPasteUsesFirstLineAndLimitsPathFields(t *testing.T) {
	for _, focus := range []configFocus{configCACertFile, configSessionDir, configCacheDir} {
		m := newTestModel(t, config.Config{}, buildinfo.Info{})
		m.dialog = dialogConfig
		m.configForm = newConfigForm(config.Config{})
		m.configForm.focus = focus

		pasted := strings.Repeat("界", maxTextFieldLength+20) + "\nignored"
		updated, _ := m.Update(tea.PasteMsg{Content: pasted})
		got := updated.(model)
		field := got.configForm.fields[int(focus)]

		if length := len([]rune(field.value)); length != maxTextFieldLength {
			t.Fatalf("focus %d pasted length = %d, want %d", focus, length, maxTextFieldLength)
		}
		if strings.Contains(field.value, "ignored") || strings.ContainsAny(field.value, "\r\n") {
			t.Fatalf("focus %d pasted value contains text after the first line: %q", focus, field.value)
		}
	}
}

func TestModel_ConfigOpensCACertificateFilePicker(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	directory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("create certificate directory: %v", err)
	}
	certificate := filepath.Join(directory, "ca.pem")
	if err := os.WriteFile(certificate, []byte("certificate"), 0o600); err != nil {
		t.Fatalf("write certificate: %v", err)
	}

	m := newTestModel(t, config.Config{Address: "localhost:8888", CACertFile: certificate}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.focus = configCACertBrowse

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if m.dialog != dialogPathPicker {
		t.Fatalf("dialog = %d, want config path picker", m.dialog)
	}
	if command == nil {
		t.Fatal("file picker init command = nil")
	}
	if m.pathPicker.currentDirectory != directory {
		t.Fatalf("picker directory = %q, want %q", m.pathPicker.currentDirectory, directory)
	}
	if m.pathPicker.rootDirectory != rootDirectory {
		t.Fatalf("picker root = %q, want %q", m.pathPicker.rootDirectory, rootDirectory)
	}

	updated, _ = m.Update(command())
	m = updated.(model)
	entry, ok := m.pathPicker.highlighted()
	if !ok || entry.name != "ca.pem" || entry.directory {
		t.Fatalf("highlighted entry = %+v, %t, want ca.pem file", entry, ok)
	}

	updated, command = m.Update(keyPress("enter"))
	m = updated.(model)
	if command != nil || m.dialog != dialogPathPicker {
		t.Fatal("Enter on a file selected it without using the Select button")
	}

	updated, _ = m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("picker focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogConfig {
		t.Fatalf("dialog after file selection = %d, want Config", got.dialog)
	}
	wantCertificate := filepath.Join("certs", "ca.pem")
	if got.configForm.fields[configCACertFile].value != wantCertificate {
		t.Fatalf("selected certificate = %q, want %q", got.configForm.fields[configCACertFile].value, wantCertificate)
	}
	if got.configForm.focus != configCACertFile {
		t.Fatalf("focus after file selection = %d, want CA cert field", got.configForm.focus)
	}
}

func TestModel_ConfigDirectoryPickerSelectsSessionDirectory(t *testing.T) {
	parent := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, parent)
	directory := filepath.Join(parent, "sessions")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatalf("create session directory: %v", err)
	}

	m := newTestModel(t, config.Config{Address: "localhost:8888", CACertFile: "/tmp/ca.pem"}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.fields[configSessionDir].setValue("sessions")
	m.configForm.focus = configSessionBrowse

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("directory picker init command = nil")
	}
	updated, _ = m.Update(command())
	m = updated.(model)
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("picker focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)

	if got.configForm.fields[configSessionDir].value != "sessions" {
		t.Fatalf("selected session directory = %q, want %q", got.configForm.fields[configSessionDir].value, "sessions")
	}
	if got.configForm.focus != configSessionDir {
		t.Fatalf("focus after directory selection = %d, want Session dir", got.configForm.focus)
	}
}

func TestNewConfigPathPicker_PreselectsConfiguredDirectory(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	localDirectory := filepath.Join(rootDirectory, ".local")
	cacheDirectory := filepath.Join(localDirectory, "cache")
	if err := os.MkdirAll(cacheDirectory, 0o700); err != nil {
		t.Fatalf("create cache directory: %v", err)
	}

	picker, command := newPathPicker(
		pathPickerCacheDirectory,
		filepath.Join(".local", "cache"),
		32,
	)
	if picker.currentDirectory != localDirectory {
		t.Fatalf("picker directory = %q, want parent %q", picker.currentDirectory, localDirectory)
	}
	picker.applyReadResult(command().(pathPickerReadMsg))
	entry, ok := picker.highlighted()
	if !ok || entry.name != "cache" || !entry.directory {
		t.Fatalf("highlighted entry = %+v, %t, want configured cache directory", entry, ok)
	}
	want := filepath.Join(".local", "cache")
	if got := picker.selectedValue(); got != want {
		t.Fatalf("selected value = %q, want %q", got, want)
	}
}

func TestConfigPathPicker_UpStopsAtLaunchDirectory(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker, _ := newPathPicker(
		pathPickerCACert,
		filepath.Join(childDirectory, "ca.pem"),
		32,
	)
	command := picker.up()
	if command == nil {
		t.Fatal("up command below picker root is nil")
	}
	if picker.currentDirectory != rootDirectory {
		t.Fatalf("current directory after up = %q, want root %q", picker.currentDirectory, rootDirectory)
	}

	picker.applyReadResult(command().(pathPickerReadMsg))
	entry, ok := picker.highlighted()
	if !ok || entry.name != "certs" {
		t.Fatalf("highlighted entry after up = %+v, %t, want certs", entry, ok)
	}

	if command = picker.up(); command != nil {
		t.Fatal("up command at picker root is not nil")
	}
	if picker.currentDirectory != rootDirectory {
		t.Fatalf("current directory escaped root: %q", picker.currentDirectory)
	}
}

func TestModel_ConfigPathPickerParentEntryMovesToParent(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker, readCommand := newPathPicker(
		pathPickerCACert,
		filepath.Join(childDirectory, "ca.pem"),
		32,
	)
	picker.applyReadResult(readCommand().(pathPickerReadMsg))
	for index, entry := range picker.entries {
		if entry.parent {
			picker.selected = index
			break
		}
	}
	entry, ok := picker.highlighted()
	if !ok || !entry.parent || entry.name != ".." {
		t.Fatalf("highlighted entry = %+v, %t, want parent entry", entry, ok)
	}

	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogPathPicker
	m.pathPicker = picker

	updated, command := m.Update(keyPress("enter"))
	got := updated.(model)
	if command == nil {
		t.Fatal("parent entry did not start loading the parent directory")
	}
	if got.pathPicker.currentDirectory != rootDirectory {
		t.Fatalf("current directory after parent entry = %q, want %q", got.pathPicker.currentDirectory, rootDirectory)
	}
}

func TestModel_ConfigPathPickerLeftRightAndSDoNotNavigate(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, rootDirectory)
	childDirectory := filepath.Join(rootDirectory, "certs")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	for _, key := range []string{"left", "right", "s"} {
		picker, _ := newPathPicker(
			pathPickerCACert,
			filepath.Join(childDirectory, "ca.pem"),
			32,
		)
		m := newTestModel(t, config.Config{}, buildinfo.Info{})
		m.dialog = dialogPathPicker
		m.pathPicker = picker

		updated, command := m.Update(keyPress(key))
		got := updated.(model)
		if command != nil {
			t.Fatalf("%s unexpectedly started navigation", key)
		}
		if got.pathPicker.currentDirectory != childDirectory {
			t.Fatalf("current directory after %s = %q, want %q", key, got.pathPicker.currentDirectory, childDirectory)
		}
	}
}

func TestConfigPathPicker_DirectoryModeHidesFiles(t *testing.T) {
	directory := canonicalPath(t.TempDir())
	changeWorkingDirectory(t, directory)
	childDirectory := filepath.Join(directory, "sessions")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, "session.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	picker, command := newPathPicker(pathPickerCacheDirectory, directory, 32)
	rawMessage := command()
	message, ok := rawMessage.(pathPickerReadMsg)
	if !ok {
		t.Fatalf("read command message type = %T, want pathPickerReadMsg", rawMessage)
	}
	picker.applyReadResult(message)

	if len(picker.entries) != 1 {
		t.Fatalf("directory picker entries = %d, want 1", len(picker.entries))
	}
	if picker.entries[0].name != "sessions" || !picker.entries[0].directory {
		t.Fatalf("directory picker entry = %+v, want sessions directory", picker.entries[0])
	}
}

func TestRenderConfigPathPickerList_UsesWhiteTreeAndPaddedSelection(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target: pathPickerCACert,
		entries: []pathPickerEntry{
			{name: "certs", directory: true},
			{name: "ca.pem"},
		},
		selected: 1,
		height:   4,
	}

	view := renderPathPickerList(theme, 30, state)
	plain := ansi.Strip(view)
	if !strings.Contains(plain, "├─ certs ") || !strings.Contains(plain, "└─ ca.pem ") {
		t.Fatalf("file picker does not render compact tree with one horizontal stroke: %q", plain)
	}
	if strings.Contains(plain, ">") {
		t.Fatalf("file picker still renders a selection arrow: %q", plain)
	}
	if !strings.Contains(view, theme.configPickerFile.Render("├─")+theme.configPickerDirectory.Render(" certs ")) {
		t.Fatal("tree branch is not white or directory name is not gray")
	}
	if !strings.Contains(view, theme.configPickerSelectedFile.Render(" ca.pem ")) {
		t.Fatal("selected file does not include inverted spaces on both sides")
	}
	lines := strings.Split(view, "\n")
	wantSelectedLine := theme.configPickerFile.Render("└─") +
		theme.configPickerSelectedFile.Render(" ca.pem ") +
		theme.configPickerArea.Width(30-lipgloss.Width("└─ ca.pem ")).Render("")
	if lines[1] != wantSelectedLine {
		t.Fatalf("selected row = %q, want branch outside padded selection", lines[1])
	}
	if !strings.Contains(view, theme.configPickerArea.Width(30).Render("")) {
		t.Fatal("file picker does not render the blue list background")
	}
}

func TestRenderConfigPathPickerList_RendersDirectoryModeDirectoriesWhite(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target:  pathPickerCacheDirectory,
		entries: []pathPickerEntry{{name: "cache", directory: true}},
		height:  2,
	}

	view := renderPathPickerList(theme, 24, state)

	if !strings.Contains(view, theme.configPickerFile.Render("└─")+theme.configPickerSelectedFile.Render(" cache ")) {
		t.Fatal("directory picker does not render a white branch and padded inverted directory")
	}
}

func TestRenderConfigPathPickerButtons_ReflectFocusAndAvailability(t *testing.T) {
	theme := newTheme()
	state := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "certs", directory: true}},
		focus:   pathPickerTree,
	}

	view := renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonDisabled.Render("< Select >")) {
		t.Fatal("Select is not disabled for a directory in the file picker")
	}
	if !strings.Contains(view, theme.button.Render("< Cancel >")) {
		t.Fatal("Cancel button is missing")
	}

	state.entries = []pathPickerEntry{{name: "ca.pem"}}
	state.focus = pathPickerSelect
	view = renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonActive.Render("< Select >")) {
		t.Fatal("focused Select button is not active")
	}

	state.focus = pathPickerCancel
	view = renderPathPickerButtons(theme, 48, state)
	if !strings.Contains(view, theme.buttonActive.Render("< Cancel >")) {
		t.Fatal("focused Cancel button is not active")
	}
}

func TestConfigPathPicker_TabSkipsDisabledSelect(t *testing.T) {
	picker := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "certs", directory: true}},
		focus:   pathPickerTree,
	}

	picker.moveFocus(1)
	if picker.focus != pathPickerCancel {
		t.Fatalf("focus = %d, want Cancel after disabled Select", picker.focus)
	}
	picker.moveFocus(1)
	if picker.focus != pathPickerTree {
		t.Fatalf("focus = %d, want Tree", picker.focus)
	}

	picker.entries = []pathPickerEntry{{name: "ca.pem"}}
	picker.moveFocus(1)
	if picker.focus != pathPickerSelect {
		t.Fatalf("focus = %d, want Select", picker.focus)
	}
}

func TestConfigPathPicker_SelectedValueMatchesConfigFieldValue(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	state := pathPicker{
		target:           pathPickerSessionDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: rootDirectory,
		entries:          []pathPickerEntry{{name: "sessions", directory: true}},
	}

	want := "sessions"
	if got := state.selectedValue(); got != want {
		t.Fatalf("selected value = %q, want %q", got, want)
	}

	state.target = pathPickerCACert
	state.entries = []pathPickerEntry{{name: "certs", directory: true}}
	if got := state.selectedValue(); got != "" {
		t.Fatalf("non-selectable selected value = %q, want empty", got)
	}
}

func TestModel_ConfigPathPickerSelectsCurrentDirectoryFromParentEntry(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	childDirectory := filepath.Join(rootDirectory, "cache")
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogPathPicker
	m.pathPicker = pathPicker{
		target:           pathPickerCacheDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: childDirectory,
		entries:          []pathPickerEntry{{name: "..", directory: true, parent: true}},
		focus:            pathPickerTree,
	}

	updated, _ := m.Update(keyPress("tab"))
	m = updated.(model)
	if m.pathPicker.focus != pathPickerSelect {
		t.Fatalf("focus = %d, want Select", m.pathPicker.focus)
	}
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogConfig {
		t.Fatalf("dialog = %d, want Config", got.dialog)
	}
	want := filepath.Base(childDirectory)
	if got.configForm.fields[configCacheDir].value != want {
		t.Fatalf("selected current directory = %q, want %q", got.configForm.fields[configCacheDir].value, want)
	}
}

func TestConfigPathPicker_EnteringDirectoryKeepsSelectedValueOnParentEntry(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	childDirectory := filepath.Join(rootDirectory, "cache")
	if err := os.Mkdir(childDirectory, 0o700); err != nil {
		t.Fatalf("create child directory: %v", err)
	}

	picker := pathPicker{
		target:           pathPickerCacheDirectory,
		rootDirectory:    rootDirectory,
		currentDirectory: rootDirectory,
		entries:          []pathPickerEntry{{name: "cache", directory: true}},
	}
	before := picker.selectedValue()
	command := picker.openHighlightedDirectory()
	if command == nil {
		t.Fatal("opening selected directory returned nil command")
	}
	picker.applyReadResult(command().(pathPickerReadMsg))

	entry, ok := picker.highlighted()
	if !ok || !entry.parent {
		t.Fatalf("highlighted entry after entering directory = %+v, %t, want ..", entry, ok)
	}
	if after := picker.selectedValue(); after != before {
		t.Fatalf("selected value after entering directory = %q, want unchanged %q", after, before)
	}
}

func TestRenderConfigPathPickerWindow_ShowsValueInsertedBySelect(t *testing.T) {
	rootDirectory := canonicalPath(t.TempDir())
	state := pathPicker{
		target:           pathPickerCACert,
		rootDirectory:    rootDirectory,
		currentDirectory: filepath.Join(rootDirectory, "certs"),
		entries:          []pathPickerEntry{{name: "ca.pem"}},
		height:           1,
	}

	plain := ansi.Strip(renderPathPickerWindow(newTheme(), 76, state))
	wantPath := filepath.Join("certs", "ca.pem")
	if !strings.Contains(plain, "Path  "+wantPath) {
		t.Fatalf("picker does not show selected config value: %q", plain)
	}
	if strings.Contains(plain, rootDirectory) {
		t.Fatalf("picker exposes absolute launch path: %q", plain)
	}
	for _, removed := range []string{"Directory", "Enter Select", "Right Open", "Current"} {
		if strings.Contains(plain, removed) {
			t.Fatalf("picker still renders removed text %q: %q", removed, plain)
		}
	}
}

func TestRenderConfigPathPickerWindow_SeparatesButtonsWithBlankLine(t *testing.T) {
	state := pathPicker{
		target:  pathPickerCACert,
		entries: []pathPickerEntry{{name: "ca.pem"}},
		height:  1,
	}

	plain := ansi.Strip(renderPathPickerWindow(newTheme(), 76, state))
	lines := strings.Split(plain, "\n")
	buttonRow := -1
	for index, line := range lines {
		if strings.Contains(line, "< Select >") {
			buttonRow = index
			break
		}
	}
	if buttonRow <= 0 || strings.TrimSpace(lines[buttonRow-1]) != "" {
		t.Fatalf("buttons are not separated from picker by a blank line: %q", plain)
	}
}

func changeWorkingDirectory(t *testing.T, directory string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func TestConfigSelectedPath(t *testing.T) {
	rootDirectory := filepath.Join(string(filepath.Separator), "tmp", "project")
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "inside root",
			path: filepath.Join(rootDirectory, ".local", "cache"),
			want: filepath.Join(".local", "cache"),
		},
		{
			name: "root directory",
			path: rootDirectory,
			want: ".",
		},
		{
			name: "outside root",
			path: filepath.Join(filepath.Dir(rootDirectory), "other", "cache"),
			want: filepath.Join(filepath.Dir(rootDirectory), "other", "cache"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := selectedPathValue(rootDirectory, test.path); got != test.want {
				t.Fatalf("selected path = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRenderConfigWindow_UsesCheckboxTransportDesign(t *testing.T) {
	view := ansi.Strip(renderConfigWindow(newTheme(), 82, newConfigForm(config.Config{
		Transport:   config.TransportGRPC,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
	}), ""))

	for _, want := range []string{"Transport", "[ ] HTTPS", "[X] gRPC", "HTTPS address", "gRPC address"} {
		if !strings.Contains(view, want) {
			t.Fatalf("Config view does not contain %q:\n%s", want, view)
		}
	}
}

func TestModel_ConfigTransportKeyboardSelection(t *testing.T) {
	m := newTestModel(t, config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
	}, buildinfo.Info{})
	m.dialog = dialogConfig
	m.configForm = newConfigForm(m.config)
	m.configForm.focus = configTransport

	updated, _ := m.Update(keyPress("right"))
	m = updated.(model)
	if m.configForm.transport != config.TransportGRPC {
		t.Fatalf("transport after Right = %q, want gRPC", m.configForm.transport)
	}

	updated, _ = m.Update(keyPress("left"))
	m = updated.(model)
	if m.configForm.transport != config.TransportHTTPS {
		t.Fatalf("transport after Left = %q, want HTTPS", m.configForm.transport)
	}

	updated, _ = m.Update(keyPress(" "))
	got := updated.(model)
	if got.configForm.transport != config.TransportGRPC {
		t.Fatalf("transport after Space = %q, want gRPC", got.configForm.transport)
	}
}

func TestModel_ConfigTransportChangePreservesOpenCacheWorkspace(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  " Alice ",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) {
			return "alice", nil
		}}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("transport change did not start session recheck")
	}
	if !m.recordFeature.workspace.open || m.recordFeature.workspace.source != recordSourceCache {
		t.Fatalf("cache workspace was closed during transport switch: %#v", m.recordFeature.workspace)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, cmd))
	got := updated.(model)
	if next != nil {
		t.Fatal("transport switch from an open cache started online record loading")
	}
	if !got.recordFeature.workspace.open || got.recordFeature.workspace.source != recordSourceCache {
		t.Fatalf("cache workspace was not preserved after session recheck: %#v", got.recordFeature.workspace)
	}
}

func TestModel_ConfigSessionChangeClosesCacheForDifferentUser(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/alice-session",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configSessionDir].setValue("/tmp/bob-session")
	m.configForm.focus = configSave

	closeCacheCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{
			currentUser: func(context.Context) (string, error) { return "bob", nil },
			closeCache:  func() { closeCacheCalls++ },
		}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("session directory change did not start session restore")
	}
	if !m.recordFeature.workspace.open || m.recordFeature.workspace.login != "alice" {
		t.Fatalf("cache workspace was closed before the new session was resolved: %#v", m.recordFeature.workspace)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if closeCacheCalls != 1 {
		t.Fatalf("cache close calls = %d, want 1", closeCacheCalls)
	}
	if !got.recordFeature.workspace.open || got.recordFeature.workspace.source != recordSourceServer {
		t.Fatalf("Bob server workspace was not opened after closing Alice cache: %#v", got.recordFeature.workspace)
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" {
		t.Fatalf("restored session = %#v, want authenticated bob", got.authentication.session)
	}
	if next == nil {
		t.Fatal("restoring another user did not start loading that user's server records")
	}
}

func TestModel_ConfigSessionChangeFailureClosesOpenCache(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/alice-session",
		CacheDir:    "/tmp/cache",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		source: recordSourceCache,
		login:  "alice",
		open:   true,
		state:  recordListReady,
	}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.fields[configSessionDir].setValue("/tmp/other-session")
	m.configForm.focus = configSave

	closeCacheCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{
			currentUser: func(context.Context) (string, error) {
				return "", errors.New("server unavailable")
			},
			closeCache: func() { closeCacheCalls++ },
		}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("session directory change did not start session restore")
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if next != nil {
		t.Fatal("failed session restore started another command")
	}
	if closeCacheCalls != 1 {
		t.Fatalf("cache close calls = %d, want 1", closeCacheCalls)
	}
	if got.recordFeature.workspace.open || got.recordFeature.workspace.source == recordSourceCache {
		t.Fatalf("cache remained open after failed restore from another session directory: %#v", got.recordFeature.workspace)
	}
	if got.authentication.session.state != authGuest {
		t.Fatalf("session state after failed restore = %d, want guest", got.authentication.session.state)
	}
}

func TestModel_ConfigTransportFailurePreservesCurrentSession(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/session",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) {
			return "", errors.New("gRPC unavailable")
		}}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	m = updated.(model)
	if command == nil {
		t.Fatal("transport change did not start session check")
	}
	if m.authentication.currentUserCheck != currentUserCheckReconfigure {
		t.Fatalf("current user check mode = %d, want reconfigure", m.authentication.currentUserCheck)
	}
	if !m.authentication.session.authenticated() || m.authentication.session.login != "alice" {
		t.Fatalf("session was cleared before transport check: %#v", m.authentication.session)
	}

	updated, next := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if next != nil {
		t.Fatal("failed transport session check started another command")
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" {
		t.Fatalf("session after transport failure = %#v, want authenticated alice", got.authentication.session)
	}
	if got.alert != alertError || got.alertTitle != "Session check failed" || got.alertReturnDialog != dialogNone {
		t.Fatalf("alert after transport failure = state %d title %q return %d", got.alert, got.alertTitle, got.alertReturnDialog)
	}
}

func TestRuntimeConfigChanged(t *testing.T) {
	https := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "https.example:8443",
		GRPCAddress: "grpc.example:9443",
		CACertFile:  "/tmp/ca.pem",
		SessionDir:  "/tmp/session",
		CacheDir:    "/tmp/cache",
	}
	grpc := https
	grpc.Transport = config.TransportGRPC

	tests := []struct {
		name      string
		previous  config.Config
		candidate config.Config
		want      bool
	}{
		{name: "unchanged HTTPS", previous: https, candidate: https},
		{name: "switch transport", previous: https, candidate: grpc, want: true},
		{name: "HTTPS active address", previous: https, candidate: withConfigAddress(https, "other.example:8443"), want: true},
		{name: "HTTPS inactive gRPC address", previous: https, candidate: withConfigGRPCAddress(https, "other.example:9443")},
		{name: "gRPC active address", previous: grpc, candidate: withConfigGRPCAddress(grpc, "other.example:9443"), want: true},
		{name: "gRPC inactive HTTPS address", previous: grpc, candidate: withConfigAddress(grpc, "other.example:8443")},
		{name: "CA certificate", previous: https, candidate: withConfigCACertFile(https, "/tmp/other-ca.pem"), want: true},
		{name: "session directory", previous: https, candidate: withConfigSessionDir(https, "/tmp/other-session"), want: true},
		{name: "cache directory", previous: https, candidate: withConfigCacheDir(https, "/tmp/other-cache"), want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := runtimeConfigChanged(test.previous, test.candidate); got != test.want {
				t.Fatalf("runtimeConfigChanged() = %t, want %t", got, test.want)
			}
		})
	}
}

func withConfigAddress(cfg config.Config, value string) config.Config {
	cfg.Address = value
	return cfg
}

func withConfigGRPCAddress(cfg config.Config, value string) config.Config {
	cfg.GRPCAddress = value
	return cfg
}

func withConfigCACertFile(cfg config.Config, value string) config.Config {
	cfg.CACertFile = value
	return cfg
}

func withConfigSessionDir(cfg config.Config, value string) config.Config {
	cfg.SessionDir = value
	return cfg
}

func withConfigCacheDir(cfg config.Config, value string) config.Config {
	cfg.CacheDir = value
	return cfg
}

func TestModel_ConfigInactiveAddressChangeDoesNotReplaceRuntime(t *testing.T) {
	initial := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "localhost:8888",
		GRPCAddress: "localhost:9090",
		SessionDir:  "/tmp/session",
	}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	closeTransportCalls := 0
	currentBackend := &transportClosingBackendStub{
		backendStub: backendStub{},
		closeTransport: func() error {
			closeTransportCalls++
			return nil
		},
	}
	m.backend = currentBackend
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.grpcAddress.setValue("grpc.example:9443")
	m.configForm.focus = configSave
	factoryCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		factoryCalls++
		return backendStub{}, nil
	}

	updated, command := m.Update(keyPress("enter"))
	got := updated.(model)
	if command != nil {
		t.Fatal("inactive gRPC address change started a session check")
	}
	if factoryCalls != 0 {
		t.Fatalf("backend factory calls = %d, want 0", factoryCalls)
	}
	if closeTransportCalls != 0 {
		t.Fatalf("active HTTPS transport close calls = %d, want 0", closeTransportCalls)
	}
	if got.backend != currentBackend {
		t.Fatal("active runtime was replaced after changing inactive gRPC address")
	}
	if got.config.GRPCAddress != "grpc.example:9443" {
		t.Fatalf("saved gRPC address = %q", got.config.GRPCAddress)
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" {
		t.Fatalf("session changed after inactive address edit: %#v", got.authentication.session)
	}
}

func TestModel_ConfigChangeClosesPreviousTransport(t *testing.T) {
	initial := config.Config{Transport: config.TransportHTTPS, Address: "localhost:8888", GRPCAddress: "localhost:9090"}
	m := newTestModel(t, initial, buildinfo.Info{})
	oldCloseCalls := 0
	oldBackend := &transportClosingBackendStub{
		backendStub: backendStub{},
		closeTransport: func() error {
			oldCloseCalls++
			return nil
		},
	}
	m.backend = oldBackend
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	m.backendFactory = func(config.Config) (Backend, error) {
		return backendStub{currentUser: func(context.Context) (string, error) { return "alice", nil }}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)
	if cmd == nil {
		t.Fatal("config change did not start session recheck")
	}
	if oldCloseCalls != 1 {
		t.Fatalf("previous transport close calls = %d, want 1", oldCloseCalls)
	}
	if got.backend == oldBackend {
		t.Fatal("backend was not replaced")
	}
}

func TestModel_ConfigSaveFailureClosesPreparedTransportAndKeepsCurrentBackend(t *testing.T) {
	initial := config.Config{Transport: config.TransportHTTPS, Address: "localhost:8888", GRPCAddress: "localhost:9090"}
	m := newTestModel(t, initial, buildinfo.Info{})
	m.configFile = "/tmp/client.json"
	m.saveConfig = func(string, config.Config) error { return errors.New("write failed") }
	m.dialog = dialogConfig
	m.configForm = newConfigForm(initial)
	m.configForm.selectTransport(config.TransportGRPC)
	m.configForm.focus = configSave
	currentBackend := &backendStub{}
	m.backend = currentBackend
	preparedCloseCalls := 0
	m.backendFactory = func(config.Config) (Backend, error) {
		return &transportClosingBackendStub{
			backendStub: backendStub{},
			closeTransport: func() error {
				preparedCloseCalls++
				return nil
			},
		}, nil
	}

	updated, cmd := m.Update(keyPress("enter"))
	got := updated.(model)
	if cmd != nil {
		t.Fatal("failed config save started a command")
	}
	if preparedCloseCalls != 1 {
		t.Fatalf("prepared transport close calls = %d, want 1", preparedCloseCalls)
	}
	if got.backend != currentBackend || got.config != initial {
		t.Fatalf("runtime changed after save failure: backend changed=%t config=%#v", got.backend != currentBackend, got.config)
	}
	if got.configForm.errorMessage != "Unable to save config file" {
		t.Fatalf("config error = %q", got.configForm.errorMessage)
	}
}
