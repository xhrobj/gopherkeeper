package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const recordsPath = "/api/v1/records"

var errRecordPayloadRequired = errors.New("record payload is required")

type recordRequest struct {
	Type    model.RecordType    `json:"type"`
	Title   string              `json:"title"`
	Payload model.RecordPayload `json:"payload"`
}

type recordMetadataResponse struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type recordResponse struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Payload   json.RawMessage  `json:"payload"`
}

type listRecordsResponse struct {
	Records []recordMetadataResponse `json:"records"`
}

// CreateRecord создаёт запись выбранного типа на Сервере.
func (c *Client) CreateRecord(
	ctx context.Context,
	accessToken string,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	body, err := newRecordRequest(title, payload)
	if err != nil {
		return model.Record{}, err
	}

	var created recordResponse
	if err := c.doJSON(ctx, jsonRequest{
		operation:      "create record",
		method:         http.MethodPost,
		path:           recordsPath,
		accessToken:    accessToken,
		requestBody:    body,
		expectedStatus: http.StatusCreated,
		responseBody:   &created,
	}); err != nil {
		return model.Record{}, err
	}

	return recordFromResponse(created)
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

// GetRecord возвращает запись текущего пользователя с payload согласно её типу.
func (c *Client) GetRecord(ctx context.Context, accessToken string, recordID string) (model.Record, error) {
	var record recordResponse

	if err := c.doJSON(ctx, jsonRequest{
		operation:      "get record",
		method:         http.MethodGet,
		path:           recordsPath + "/" + url.PathEscape(recordID),
		accessToken:    accessToken,
		expectedStatus: http.StatusOK,
		responseBody:   &record,
	}); err != nil {
		return model.Record{}, err
	}

	return recordFromResponse(record)
}

// UpdateRecord изменяет запись на Сервере с проверкой ожидаемой ревизии.
func (c *Client) UpdateRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	body, err := newRecordRequest(title, payload)
	if err != nil {
		return model.Record{}, err
	}

	var updated recordResponse
	if err := c.doJSON(ctx, jsonRequest{
		operation:   "update record",
		method:      http.MethodPut,
		path:        recordsPath + "/" + url.PathEscape(recordID),
		accessToken: accessToken,
		headers: map[string]string{
			"If-Match": recordRevisionETag(expectedRevision),
		},
		requestBody:    body,
		expectedStatus: http.StatusOK,
		responseBody:   &updated,
	}); err != nil {
		return model.Record{}, err
	}

	return recordFromResponse(updated)
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

func newRecordRequest(title string, payload model.RecordPayload) (recordRequest, error) {
	if payload == nil {
		return recordRequest{}, errRecordPayloadRequired
	}
	return recordRequest{
		Type:    payload.RecordType(),
		Title:   title,
		Payload: payload,
	}, nil
}

func recordFromResponse(response recordResponse) (model.Record, error) {
	payload, err := model.NewRecordPayload(response.Type)
	if err != nil {
		return model.Record{}, fmt.Errorf("select record payload: %w", err)
	}
	if err := decodeRecordPayload(response.Payload, payload); err != nil {
		return model.Record{}, fmt.Errorf("decode record payload: %w", err)
	}
	if err := payload.Validate(); err != nil {
		return model.Record{}, fmt.Errorf("validate record payload: %w", err)
	}

	return model.Record{
		Metadata: recordMetadataFromResponse(recordMetadataResponse{
			ID:        response.ID,
			Type:      response.Type,
			Title:     response.Title,
			Revision:  response.Revision,
			CreatedAt: response.CreatedAt,
			UpdatedAt: response.UpdatedAt,
		}),
		Payload: payload,
	}, nil
}

func decodeRecordPayload(data json.RawMessage, payload model.RecordPayload) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(payload); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("record payload must contain one JSON value")
		}

		return err
	}

	return nil
}

func recordRevisionETag(revision int64) string {
	return strconv.Quote(strconv.FormatInt(revision, 10))
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
