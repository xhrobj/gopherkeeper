package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func recordViewLines(t theme, state recordViewState, width int) []recordViewLine {
	record := state.record
	metadata := record.Metadata

	lines := make([]recordViewLine, 0, 16)
	lines = append(lines, recordViewMetadataLines(metadata, width)...)
	lines = append(lines, recordViewLine{})
	lines = appendRecordViewField(lines, "Title", metadata.Title, width)
	lines = append(lines, recordViewLine{})
	lines = append(lines, recordViewPayloadLines(t, state, width)...)

	return lines
}

func recordViewPayloadLines(t theme, state recordViewState, width int) []recordViewLine {
	switch payload := state.record.Payload.(type) {
	case *recordmodel.TextPayload:
		return recordViewTextPayloadLines(t, state, payload, width)
	case *recordmodel.CredentialsPayload:
		return recordViewCredentialsPayloadLines(payload, state.revealed, width)
	case *recordmodel.CardPayload:
		return recordViewCardPayloadLines(t, payload, state.revealed, width)
	case *recordmodel.BinaryPayload:
		return recordViewBinaryPayloadLines(payload, width)
	default:
		return []recordViewLine{{message: "Unsupported record payload"}}
	}
}

func recordViewTextPayloadLines(
	t theme,
	state recordViewState,
	payload *recordmodel.TextPayload,
	width int,
) []recordViewLine {
	if payload == nil {
		return nil
	}

	lines := make([]recordViewLine, 0, 4)

	if state.textArea.active() {
		lines = append(lines, recordViewLine{label: "Text:", textArea: true})
		for _, row := range renderReadOnlyTextArea(t, state.textArea, width) {
			lines = append(lines, recordViewLine{rendered: row, textArea: true})
		}
	} else {
		lines = appendRecordViewField(lines, "Text", payload.Text, width)
	}

	return appendRecordViewNotes(lines, payload.Metadata, width)
}

func recordViewCredentialsPayloadLines(
	payload *recordmodel.CredentialsPayload,
	reveal bool,
	width int,
) []recordViewLine {
	if payload == nil {
		return nil
	}

	lines := make([]recordViewLine, 0, 4)
	lines = appendRecordViewField(lines, "Login", payload.Login, width)
	lines = appendRecordViewField(lines, "Password", visibleSecret(payload.Password, reveal), width)

	if payload.URL != "" {
		lines = appendRecordViewField(lines, "URL", payload.URL, width)
	}

	return appendRecordViewNotes(lines, payload.Metadata, width)
}

func recordViewCardPayloadLines(
	t theme,
	payload *recordmodel.CardPayload,
	reveal bool,
	width int,
) []recordViewLine {
	if payload == nil {
		return nil
	}

	lines := buildCardPreviewLines(t, payload, reveal, width)

	return appendRecordViewNotes(lines, payload.Metadata, width)
}

func recordViewBinaryPayloadLines(payload *recordmodel.BinaryPayload, width int) []recordViewLine {
	if payload == nil {
		return nil
	}

	lines := make([]recordViewLine, 0, 3)
	lines = appendRecordViewField(lines, "Filename", payload.Filename, width)
	lines = appendRecordViewField(lines, "Size", fmt.Sprintf("%d bytes", len(payload.Data)), width)

	return appendRecordViewNotes(lines, payload.Metadata, width)
}

func appendRecordViewNotes(lines []recordViewLine, notes string, width int) []recordViewLine {
	lines = append(lines, recordViewLine{})

	return appendRecordViewField(lines, "Notes", notes, width)
}

func recordViewMetadataLines(metadata recordmodel.RecordMetadata, width int) []recordViewLine {
	lines := make([]recordViewLine, 0, 5)
	lines = appendRecordViewField(lines, "ID", metadata.ID, width)
	lines = appendRecordViewField(lines, "Revision", fmt.Sprintf("%d", metadata.Revision), width)
	lines = append(lines, recordViewLine{})
	lines = appendRecordViewField(lines, "Created at", formatRecordTime(metadata.CreatedAt), width)
	lines = appendRecordViewField(lines, "Updated at", formatRecordTime(metadata.UpdatedAt), width)

	return lines
}

func appendRecordViewField(lines []recordViewLine, label, value string, width int) []recordViewLine {
	prefix := label + ": "
	prefixWidth := lipgloss.Width(prefix)
	valueWidth := max(1, width-prefixWidth)
	wrapped := wrapRecordViewText(value, valueWidth)

	if len(wrapped) == 0 {
		wrapped = []string{""}
	}

	lines = append(lines, recordViewLine{label: prefix, value: wrapped[0]})
	indent := strings.Repeat(" ", prefixWidth)

	for _, continuation := range wrapped[1:] {
		lines = append(lines, recordViewLine{label: indent, value: continuation})
	}

	return lines
}

func recordTypeTitle(recordType recordmodel.RecordType) string {
	value := string(recordType)

	if value == "" {
		return ""
	}

	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]

	return string(runes)
}
