package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	recordUpdatedAtLayout                = "2006-01-02 15:04"
	recordWorkspaceBodyVerticalPadding   = 1
	recordWorkspaceBodyHorizontalPadding = 2
	recordWorkspaceRowsBeforeList        = 2
	recordWorkspaceFirstRowOffset        = 1 + recordWorkspaceBodyVerticalPadding + recordWorkspaceRowsBeforeList
)

func recordWorkspaceWidth(screenWidth int) int {
	return clamp(screenWidth-6, 56, 112)
}

func recordWorkspaceHeight(screenHeight int) int {
	return max(14, screenHeight-4)
}

func recordWorkspacePageSize(screenHeight int) int {
	return recordWorkspacePageSizeForWindow(recordWorkspaceHeight(screenHeight))
}

func recordWorkspacePageSizeForWindow(height int) int {
	innerHeight := max(8, height-3)
	return max(1, innerHeight-recordWorkspaceRowsBeforeList)
}

type recordWorkspaceLayout struct {
	contentWidth int
	innerHeight  int
	pageSize     int
	rowBounds    layoutBounds
}

func newRecordWorkspaceLayout(width, height int) recordWorkspaceLayout {
	contentWidth := max(1, width-4)
	innerHeight := max(8, height-3)
	pageSize := recordWorkspacePageSizeForWindow(height)

	return recordWorkspaceLayout{
		contentWidth: contentWidth,
		innerHeight:  innerHeight,
		pageSize:     pageSize,
		rowBounds: layoutBounds{
			x:      recordWorkspaceBodyHorizontalPadding,
			y:      recordWorkspaceFirstRowOffset,
			width:  contentWidth,
			height: pageSize,
		},
	}
}

func renderRecordWorkspace(
	t theme,
	width, height int,
	workspace recordWorkspace,
	active bool,
	pending bool,
	spinnerFrame string,
) string {
	layout := newRecordWorkspaceLayout(width, height)
	contentWidth := layout.contentWidth
	innerHeight := layout.innerHeight
	pageSize := layout.pageSize

	rows := make([]string, 0, innerHeight)
	rows = append(rows, renderRecordHeader(t, contentWidth))
	rows = append(rows, renderRecordSeparator(t, contentWidth))
	rows = append(rows, renderRecordRows(t, contentWidth, pageSize, workspace)...)

	for len(rows) < innerHeight {
		rows = append(rows, t.windowBody.Width(contentWidth).Render(""))
	}
	if len(rows) > innerHeight {
		rows = rows[:innerHeight]
	}

	titleStyle := t.windowTitle
	if !active {
		titleStyle = t.windowTitleInactive
	}
	titleLine := renderWindowTitle(titleStyle, width, "Records", spinnerFrame, pending)
	body := t.windowBody.
		Width(width).
		Height(max(1, height-1)).
		Padding(recordWorkspaceBodyVerticalPadding, recordWorkspaceBodyHorizontalPadding).
		Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
}

func renderRecordHeader(t theme, width int) string {
	columns := recordColumns(width)
	return renderRecordHeaderLine(t, columns)
}

func renderRecordSeparator(t theme, width int) string {
	columns := recordColumns(width)
	line := strings.Repeat("─", columns.recordType) + "─┼─" +
		strings.Repeat("─", columns.title) + "─┼─" +
		strings.Repeat("─", columns.revision) + "─┼─" +
		strings.Repeat("─", columns.updated)

	return t.recordGrid.Width(width).Render(fitSingleLine(line, width))
}

func renderRecordRows(t theme, width, pageSize int, workspace recordWorkspace) []string {
	rows := make([]string, 0, pageSize)
	columns := recordColumns(width)

	if workspace.state == recordListLoading {
		// During loading the empty table remains visible; progress is shown in the title.
	} else if workspace.state == recordListFailed {
		message := workspace.failure
		if message == "" {
			message = "Unable to load records"
		}
		rows = append(rows, t.recordMessage.Width(width).AlignHorizontal(lipgloss.Center).Render(fitSingleLine(message, width)))
	} else if workspace.state == recordListReady && len(workspace.records) == 0 {
		rows = append(rows, t.recordMessage.Width(width).AlignHorizontal(lipgloss.Center).Render("No records found"))
	} else {
		end := min(len(workspace.records), workspace.offset+pageSize)
		for index := workspace.offset; index < end; index++ {
			metadata := workspace.records[index]
			rows = append(rows, renderRecordMetadataLine(t, columns, metadata, index == workspace.selected))
		}
	}

	for len(rows) < pageSize {
		rows = append(rows, t.recordRow.Width(width).Render(""))
	}

	return rows[:pageSize]
}

type recordColumnWidths struct {
	recordType int
	title      int
	revision   int
	updated    int
}

func recordColumns(width int) recordColumnWidths {
	const (
		typeWidth     = 12
		revisionWidth = 5
		updatedWidth  = 16
		gapsWidth     = 9
	)

	return recordColumnWidths{
		recordType: typeWidth,
		title:      max(8, width-typeWidth-revisionWidth-updatedWidth-gapsWidth),
		revision:   revisionWidth,
		updated:    updatedWidth,
	}
}

func renderRecordMetadataLine(
	t theme,
	columns recordColumnWidths,
	metadata recordmodel.RecordMetadata,
	selected bool,
) string {
	cellStyle := t.recordRow
	separatorStyle := t.recordGrid
	typeStyle := cellStyle

	if selected {
		cellStyle = t.recordSelected
		separatorStyle = t.recordSelected
		typeStyle = t.recordSelected
	}

	separator := separatorStyle.Render(" │ ")

	return renderRecordCell(typeStyle, string(metadata.Type), columns.recordType, lipgloss.Left) + separator +
		renderRecordCell(cellStyle, metadata.Title, columns.title, lipgloss.Left) + separator +
		renderRecordCell(cellStyle, fmt.Sprintf("%d", metadata.Revision), columns.revision, lipgloss.Right) + separator +
		renderRecordCell(cellStyle, formatRecordTime(metadata.UpdatedAt), columns.updated, lipgloss.Left)
}

func renderRecordHeaderLine(t theme, columns recordColumnWidths) string {
	separator := t.recordGrid.Render(" │ ")

	return renderRecordCell(t.recordHeader, "TYPE", columns.recordType, lipgloss.Left) + separator +
		renderRecordCell(t.recordHeader, "TITLE", columns.title, lipgloss.Left) + separator +
		renderRecordCell(t.recordHeader, "REV", columns.revision, lipgloss.Right) + separator +
		renderRecordCell(t.recordHeader, "UPDATED", columns.updated, lipgloss.Left)
}

func renderRecordCell(style lipgloss.Style, value string, width int, align lipgloss.Position) string {
	return style.Width(width).AlignHorizontal(align).Render(fitSingleLine(value, width))
}

func formatRecordTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Local().Format(recordUpdatedAtLayout)
}
