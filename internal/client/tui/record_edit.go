package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordEditStatus int

const (
	recordEditLoading recordEditStatus = iota + 1
	recordEditReady
)

type recordEditState struct {
	status       recordEditStatus
	record       recordmodel.Record
	form         recordForm
	returnDialog dialogID
}

func (state *recordEditState) begin(metadata recordmodel.RecordMetadata, returnDialog dialogID) {
	state.status = recordEditLoading
	state.record = recordmodel.Record{Metadata: metadata}
	state.form = newRecordEditForm(state.record)
	state.returnDialog = returnDialog
}

func (state *recordEditState) apply(record recordmodel.Record, returnDialog dialogID) {
	state.status = recordEditReady
	state.record = record
	state.form = newRecordEditForm(record)
	state.returnDialog = returnDialog
}

func (state *recordEditState) clear() {
	*state = recordEditState{}
}

type recordEditLoadResultMsg struct {
	requestID uint64
	record    recordmodel.Record
	err       error
}

type recordEditResultMsg struct {
	requestID uint64
	record    recordmodel.Record
	err       error
}

func recordEditLoadCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	recordID string,
) tea.Cmd {
	return func() tea.Msg {
		record, err := backend.GetRecord(ctx, recordID)
		return recordEditLoadResultMsg{requestID: requestID, record: record, err: err}
	}
}

func recordEditCommand(
	ctx context.Context,
	backend Backend,
	readBinary binaryFileReader,
	requestID uint64,
	record recordmodel.Record,
	input recordFormInput,
) tea.Cmd {
	return func() tea.Msg {
		payload, err := input.buildPayload(readBinary, record.Payload)
		if err != nil {
			return recordEditResultMsg{requestID: requestID, err: err}
		}

		updated, err := backend.UpdateRecord(
			ctx,
			record.Metadata.ID,
			record.Metadata.Revision,
			input.title,
			payload,
		)

		return recordEditResultMsg{requestID: requestID, record: updated, err: err}
	}
}

func cleanRecordEditError(err error) string {
	return cleanRecordMutationError(err, recordMutationUpdate)
}

func recordEditNotice(record recordmodel.Record) string {
	return fmt.Sprintf("Updated %s record %q", record.Metadata.Type, record.Metadata.Title)
}
