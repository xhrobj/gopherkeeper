package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestReadOnlyTextArea_ScrollsWrappedText(t *testing.T) {
	area := readOnlyTextArea{}
	area.resize(8, 3)
	area.setValue("first line\nsecond line\nthird line\nfourth line")
	if area.maxOffset() == 0 {
		t.Fatal("long text area is not scrollable")
	}
	first := strings.Join(area.visibleLines(), "|")
	if !area.scroll(1) || area.offset != 1 {
		t.Fatalf("scroll state = offset %d", area.offset)
	}
	second := strings.Join(area.visibleLines(), "|")
	if first == second {
		t.Fatalf("visible text did not change after scroll: %q", first)
	}
	area.end()
	if area.offset != area.maxOffset() {
		t.Fatalf("End offset = %d, want %d", area.offset, area.maxOffset())
	}
	area.home()
	if area.offset != 0 {
		t.Fatalf("Home offset = %d", area.offset)
	}
}

func TestWrapRecordViewText_NormalizesTabsAndCarriageReturns(t *testing.T) {
	got := strings.Join(wrapRecordViewText("one\ttwo\rthree", 40), "|")
	if got != "one    two|three" {
		t.Fatalf("normalized text = %q", got)
	}
}

func TestRecordView_TextAlwaysUsesReadOnlyTextArea(t *testing.T) {
	for _, value := range []string{"", "short text", strings.Repeat("x", 256)} {
		state := recordViewState{}
		state.apply(recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Note", Revision: 1},
			Payload:  &recordmodel.TextPayload{Text: value, Metadata: "notes remain visible"},
		}, 80, 30)
		if !state.textArea.active() {
			t.Fatalf("text %q does not use multiline viewport", value)
		}
		plain := ansi.Strip(renderRecordViewWindow(newTheme(), 66, 36, state, 0, false, false, ""))
		for _, want := range []string{"Text:", "▲", "▼", "Notes: notes remain visible", "< Close >"} {
			if !strings.Contains(plain, want) {
				t.Fatalf("text view does not contain %q:\n%s", want, plain)
			}
		}
		for _, unwanted := range []string{"┌", "┐", "└", "┘", "│", "─"} {
			if strings.Contains(plain, unwanted) {
				t.Fatalf("text view still contains ASCII frame glyph %q:\n%s", unwanted, plain)
			}
		}
	}
}

func TestModel_RecordViewScrollsLongTextBeforeOuterWindow(t *testing.T) {
	text := strings.Repeat("line of text\n", 40)
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 42
	m.dialog = dialogRecordView
	m.recordFeature.view.apply(recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Long note", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: text},
	}, m.width, m.height)

	beforeOuter := m.recordFeature.view.offset
	updated, _ := m.updateRecordView("down")
	got := updated.(model)
	if got.recordFeature.view.textArea.offset != 1 {
		t.Fatalf("text area offset = %d, want 1", got.recordFeature.view.textArea.offset)
	}
	if got.recordFeature.view.offset != beforeOuter {
		t.Fatalf("outer offset changed from %d to %d while text area was visible", beforeOuter, got.recordFeature.view.offset)
	}
}

func TestModel_RecordViewMouseWheelScrollsTextArea(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 42
	m.dialog = dialogRecordView
	m.recordFeature.view.apply(recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Long note", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: strings.Repeat("line of text\n", 40)},
	}, m.width, m.height)
	bounds, ok := m.recordViewTextAreaBounds()
	if !ok {
		t.Fatal("text area bounds were not found")
	}

	message := tea.MouseWheelMsg(tea.Mouse{
		X: bounds.x + 1, Y: bounds.y + 1, Button: tea.MouseWheelDown,
	})
	updated, _ := m.Update(message)
	got := updated.(model)
	if got.recordFeature.view.textArea.offset != readOnlyTextAreaWheelStep {
		t.Fatalf("wheel offset = %d, want %d", got.recordFeature.view.textArea.offset, readOnlyTextAreaWheelStep)
	}
}
