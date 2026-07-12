package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type recordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	payload          model.RecordPayload
}

func executeCreateRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	title string,
	payload model.RecordPayload,
) error {
	record, err := application.CreateRecord(ctx, usecase.CreateRecordRequest{
		Title:   title,
		Payload: payload,
	})
	if err != nil {
		return err
	}

	recordType := payload.RecordType()
	if _, err := fmt.Fprintf(
		output,
		"Created %s record %s with revision %d.\n",
		recordType,
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write created %s record: %w", recordType, err)
	}

	return nil
}

func executeUpdateRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	request recordUpdateCommandRequest,
) error {
	record, err := application.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID:         request.recordID,
		ExpectedRevision: request.expectedRevision,
		Title:            request.title,
		Payload:          request.payload,
	})
	if err != nil {
		return err
	}

	recordType := request.payload.RecordType()
	if _, err := fmt.Fprintf(
		output,
		"Updated %s record %s to revision %d.\n",
		recordType,
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write updated %s record: %w", recordType, err)
	}

	return nil
}
