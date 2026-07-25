package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type binaryFileReader func(string) (string, []byte, error)

type recordCreateResultMsg struct {
	requestID uint64
	record    recordmodel.Record
	err       error
}

func recordCreateCommand(
	ctx context.Context,
	backend Backend,
	readBinary binaryFileReader,
	requestID uint64,
	input recordFormInput,
) tea.Cmd {
	return func() tea.Msg {
		payload, err := input.buildPayload(readBinary, nil)
		if err != nil {
			return recordCreateResultMsg{requestID: requestID, err: err}
		}

		record, err := backend.CreateRecord(ctx, input.title, payload)

		return recordCreateResultMsg{requestID: requestID, record: record, err: err}
	}
}

func cleanRecordCreateError(err error) string {
	return cleanRecordMutationError(err, recordMutationCreate)
}

func recordCreateNotice(record recordmodel.Record) string {
	return fmt.Sprintf("Created %s record %q", record.Metadata.Type, record.Metadata.Title)
}
