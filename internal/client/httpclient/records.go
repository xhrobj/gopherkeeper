package httpclient

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const recordsPath = "/api/v1/records"

// CreateTextRecordRequest содержит данные для создания text-записи через HTTP API.
type CreateTextRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Payload содержит приватный text payload.
	Payload model.TextPayload
}

// UpdateTextRecordRequest содержит данные для изменения text-записи через HTTP API.
type UpdateTextRecordRequest struct {
	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новый приватный text payload.
	Payload model.TextPayload
}

// TextRecord содержит открытую metadata и расшифрованный text payload, возвращённые Сервером.
type TextRecord struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный text payload.
	Payload model.TextPayload
}

type createRecordRequest struct {
	Type    model.RecordType  `json:"type"`
	Title   string            `json:"title"`
	Payload model.TextPayload `json:"payload"`
}

type updateRecordRequest struct {
	Type    model.RecordType  `json:"type"`
	Title   string            `json:"title"`
	Payload model.TextPayload `json:"payload"`
}

type recordMetadataResponse struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type textRecordResponse struct {
	ID        string            `json:"id"`
	Type      model.RecordType  `json:"type"`
	Title     string            `json:"title"`
	Revision  int64             `json:"revision"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Payload   model.TextPayload `json:"payload"`
}

type listRecordsResponse struct {
	Records []recordMetadataResponse `json:"records"`
}

// CreateTextRecord создаёт text-запись на Сервере.
func (c *Client) CreateTextRecord(
	ctx context.Context,
	accessToken string,
	request CreateTextRecordRequest,
) (TextRecord, error) {
	var created textRecordResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:   "create text record",
		method:      http.MethodPost,
		path:        recordsPath,
		accessToken: accessToken,
		requestBody: createRecordRequest{
			Type:    model.RecordTypeText,
			Title:   request.Title,
			Payload: request.Payload,
		},
		expectedStatus: http.StatusCreated,
		responseBody:   &created,
	}); err != nil {
		return TextRecord{}, err
	}

	return textRecordFromResponse(created), nil
}

// ListRecords возвращает metadata приватных записей текущего пользователя.
func (c *Client) ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error) {
	var listed listRecordsResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:      "list records",
		method:         http.MethodGet,
		path:           recordsPath,
		accessToken:    accessToken,
		expectedStatus: http.StatusOK,
		responseBody:   &listed,
	}); err != nil {
		return nil, err
	}

	records := make([]model.RecordMetadata, 0, len(listed.Records))
	for _, item := range listed.Records {
		records = append(records, recordMetadataFromResponse(item))
	}

	return records, nil
}

// GetTextRecord возвращает text-запись текущего пользователя.
func (c *Client) GetTextRecord(ctx context.Context, accessToken string, recordID string) (TextRecord, error) {
	var record textRecordResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:      "get text record",
		method:         http.MethodGet,
		path:           recordsPath + "/" + url.PathEscape(recordID),
		accessToken:    accessToken,
		expectedStatus: http.StatusOK,
		responseBody:   &record,
	}); err != nil {
		return TextRecord{}, err
	}

	return textRecordFromResponse(record), nil
}

// UpdateTextRecord изменяет text-запись на Сервере с проверкой ожидаемой ревизии.
func (c *Client) UpdateTextRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
	request UpdateTextRecordRequest,
) (TextRecord, error) {
	var updated textRecordResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:   "update text record",
		method:      http.MethodPut,
		path:        recordsPath + "/" + url.PathEscape(recordID),
		accessToken: accessToken,
		headers: map[string]string{
			"If-Match": recordRevisionETag(expectedRevision),
		},
		requestBody: updateRecordRequest{
			Type:    model.RecordTypeText,
			Title:   request.Title,
			Payload: request.Payload,
		},
		expectedStatus: http.StatusOK,
		responseBody:   &updated,
	}); err != nil {
		return TextRecord{}, err
	}

	return textRecordFromResponse(updated), nil
}

// DeleteRecord удаляет запись на Сервере с проверкой ожидаемой ревизии.
func (c *Client) DeleteRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
) error {
	return c.doJSON(ctx, jsonRequest{
		operation:   "delete record",
		method:      http.MethodDelete,
		path:        recordsPath + "/" + url.PathEscape(recordID),
		accessToken: accessToken,
		headers: map[string]string{
			"If-Match": recordRevisionETag(expectedRevision),
		},
		expectedStatus: http.StatusNoContent,
	})
}

func recordRevisionETag(revision int64) string {
	return strconv.Quote(strconv.FormatInt(revision, 10))
}

func textRecordFromResponse(response textRecordResponse) TextRecord {
	return TextRecord{
		Metadata: recordMetadataFromResponse(recordMetadataResponse{
			ID:        response.ID,
			Type:      response.Type,
			Title:     response.Title,
			Revision:  response.Revision,
			CreatedAt: response.CreatedAt,
			UpdatedAt: response.UpdatedAt,
		}),
		Payload: response.Payload,
	}
}

func recordMetadataFromResponse(response recordMetadataResponse) model.RecordMetadata {
	return model.RecordMetadata{
		ID:        response.ID,
		Type:      response.Type,
		Title:     response.Title,
		Revision:  response.Revision,
		CreatedAt: response.CreatedAt,
		UpdatedAt: response.UpdatedAt,
	}
}
