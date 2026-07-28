package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

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

func TestConfigWindowLayout_UsesRenderedControlGeometry(t *testing.T) {
	theme := newTheme()
	cfg := config.Config{
		Transport:   config.TransportHTTPS,
		Address:     "https.example:8443",
		GRPCAddress: "grpc.example:9090",
		CACertFile:  "certs/ca.pem",
		SessionDir:  "session-dir",
		CacheDir:    "cache-dir",
	}
	form := newConfigForm(cfg)
	form.focus = configSave
	layout := newConfigWindowLayout(theme, 82, form, "configs/client.json")
	lines := strings.Split(ansi.Strip(layout.content), "\n")

	assertConfigTransportGeometry(t, lines, layout.transportBounds)
	assertConfigFieldGeometry(t, lines, layout.fieldBounds, cfg)
	assertConfigBrowseGeometry(t, lines, layout.browseBounds, layout.fieldBounds)
	assertConfigButtonGeometry(t, lines, layout.buttonBounds)
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
