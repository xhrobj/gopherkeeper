package cli

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func writeRecordList(output io.Writer, records []model.RecordMetadata) error {
	if len(records) == 0 {
		if _, err := fmt.Fprintln(output, "No records found."); err != nil {
			return fmt.Errorf("write empty record list: %w", err)
		}

		return nil
	}

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tTYPE\tTITLE\tREVISION\tUPDATED AT"); err != nil {
		return fmt.Errorf("write record list header: %w", err)
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%s\n",
			record.ID,
			record.Type,
			record.Title,
			record.Revision,
			formatRecordTime(record.UpdatedAt),
		); err != nil {
			return fmt.Errorf("write record list item: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush record list: %w", err)
	}

	return nil
}

func writeRecord(output io.Writer, record model.Record, outputPath string) error {
	if record.Metadata.Type == model.RecordTypeBinary {
		return writeBinaryRecord(output, record, outputPath)
	}
	if outputPath != "" {
		return errors.New("--output can only be used with binary records")
	}

	return writeNonBinaryRecord(output, record)
}

func writeRecordHeader(output io.Writer, metadata model.RecordMetadata) error {
	if _, err := fmt.Fprintf(
		output,
		"ID: %s\nType: %s\nTitle: %s\nRevision: %d\nCreated at: %s\nUpdated at: %s\n",
		metadata.ID,
		metadata.Type,
		metadata.Title,
		metadata.Revision,
		formatRecordTime(metadata.CreatedAt),
		formatRecordTime(metadata.UpdatedAt),
	); err != nil {
		return fmt.Errorf("write record metadata: %w", err)
	}

	return nil
}
func writeBinaryRecordPayload(output io.Writer, payload *model.BinaryPayload, outputPath string) error {
	if _, err := fmt.Fprintf(
		output,
		"\nFilename: %s\nSize: %d bytes\nSaved to: %s\n",
		payload.Filename,
		len(payload.Data),
		outputPath,
	); err != nil {
		return fmt.Errorf("write binary record: %w", err)
	}

	if payload.ContentType != "" {
		if _, err := fmt.Fprintf(output, "Content type: %s\n", payload.ContentType); err != nil {
			return fmt.Errorf("write binary record content type: %w", err)
		}
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "binary record")
}
func writeTextRecordPayload(output io.Writer, payload *model.TextPayload) error {
	if _, err := fmt.Fprintf(output, "\nText:\n%s\n", payload.Text); err != nil {
		return fmt.Errorf("write text record: %w", err)
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "text record")
}
func writeCredentialsRecordPayload(output io.Writer, payload *model.CredentialsPayload) error {
	if _, err := fmt.Fprintf(
		output,
		"\nLogin: %s\nPassword: %s\n",
		payload.Login,
		payload.Password,
	); err != nil {
		return fmt.Errorf("write credentials record: %w", err)
	}

	if payload.URL != "" {
		if _, err := fmt.Fprintf(output, "URL: %s\n", payload.URL); err != nil {
			return fmt.Errorf("write credentials record URL: %w", err)
		}
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "credentials record")
}
func writeCardRecordPayload(output io.Writer, payload *model.CardPayload) error {
	if _, err := fmt.Fprintf(output, "\nNumber: %s\n", payload.Number); err != nil {
		return fmt.Errorf("write card record: %w", err)
	}

	if payload.Cardholder != "" {
		if _, err := fmt.Fprintf(output, "Cardholder: %s\n", payload.Cardholder); err != nil {
			return fmt.Errorf("write cardholder: %w", err)
		}
	}

	if payload.ExpiryMonth != nil && payload.ExpiryYear != nil {
		if _, err := fmt.Fprintf(
			output,
			"Expiry: %02d/%d\n",
			*payload.ExpiryMonth,
			*payload.ExpiryYear,
		); err != nil {
			return fmt.Errorf("write card expiry: %w", err)
		}
	}

	if payload.CVV != "" {
		if _, err := fmt.Fprintf(output, "CVV: %s\n", payload.CVV); err != nil {
			return fmt.Errorf("write card CVV: %w", err)
		}
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "card record")
}
func writeRecordMetadataPayload(output io.Writer, metadata string, description string) error {
	if metadata == "" {
		return nil
	}

	if _, err := fmt.Fprintf(output, "\nMetadata:\n%s\n", metadata); err != nil {
		return fmt.Errorf("write %s metadata: %w", description, err)
	}

	return nil
}
func formatRecordTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
