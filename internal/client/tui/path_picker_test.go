package tui

import (
	"path/filepath"
	"testing"
)

func TestSelectedPathValue_ReturnsRelativePathInsideRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "workspace")
	path := filepath.Join(root, "certs", "ca.pem")

	got := selectedPathValue(root, path)
	want := filepath.Join("certs", "ca.pem")
	if got != want {
		t.Fatalf("selectedPathValue() = %q, want %q", got, want)
	}
}

func TestSelectedPathValue_KeepsPathOutsideRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "workspace")
	path := filepath.Join(string(filepath.Separator), "private", "session.json")

	if got := selectedPathValue(root, path); got != path {
		t.Fatalf("selectedPathValue() = %q, want %q", got, path)
	}
}

func TestSelectedPathValue_CleansPaths(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "workspace", ".")
	path := filepath.Join(root, "cache", "..", "cache")

	if got := selectedPathValue(root, path); got != "cache" {
		t.Fatalf("selectedPathValue() = %q, want cache", got)
	}
}

func TestConfigBrowseTarget_MapsBrowseFields(t *testing.T) {
	tests := []struct {
		focus configFocus
		want  pathPickerTarget
		ok    bool
	}{
		{focus: configCACertBrowse, want: pathPickerCACert, ok: true},
		{focus: configSessionBrowse, want: pathPickerSessionDirectory, ok: true},
		{focus: configCacheBrowse, want: pathPickerCacheDirectory, ok: true},
		{focus: configAddress, ok: false},
	}

	for _, test := range tests {
		got, ok := configBrowseTarget(test.focus)
		if got != test.want || ok != test.ok {
			t.Fatalf("configBrowseTarget(%d) = (%d, %v), want (%d, %v)", test.focus, got, ok, test.want, test.ok)
		}
	}
}

func TestConfigTargetFieldHelpers_ReturnConfiguredField(t *testing.T) {
	if got := configTargetFieldFocus(pathPickerSessionDirectory); got != configSessionDir {
		t.Fatalf("configTargetFieldFocus() = %d, want %d", got, configSessionDir)
	}
	if got := configTargetFieldIndex(pathPickerCacheDirectory); got != 3 {
		t.Fatalf("configTargetFieldIndex() = %d, want 3", got)
	}
}

func TestNewPathPickerWindowLayout_ClampsDimensions(t *testing.T) {
	layout := newPathPickerWindowLayout(3, 0)
	if layout.contentWidth != 1 {
		t.Fatalf("content width = %d, want 1", layout.contentWidth)
	}
	if layout.listBounds != (layoutBounds{x: 2, y: 4, width: 1, height: 1}) {
		t.Fatalf("list bounds = %#v", layout.listBounds)
	}
}

func TestPathPickerSizeHelpers_ClampScreenDimensions(t *testing.T) {
	if got := pathPickerListHeight(1); got != 6 {
		t.Fatalf("pathPickerListHeight(1) = %d, want 6", got)
	}
	if got := pathPickerListHeight(100); got != 14 {
		t.Fatalf("pathPickerListHeight(100) = %d, want 14", got)
	}
	if got := pathPickerWindowWidth(1); got != 54 {
		t.Fatalf("pathPickerWindowWidth(1) = %d, want 54", got)
	}
	if got := pathPickerWindowWidth(200); got != 82 {
		t.Fatalf("pathPickerWindowWidth(200) = %d, want 82", got)
	}
}
