package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	"github.com/xhrobj/gopherkeeper/internal/server/transport/http/middleware"
)

const testRecordID = "7b4c2d7d-0e2f-4c4b-8d4b-8f4f7c4d3a21"

type recordManagerStub struct {
	create func(context.Context, service.CreateRecordRequest) (model.Record, error)
	list   func(context.Context, int64) ([]model.RecordMetadata, error)
	get    func(context.Context, int64, string) (model.Record, error)
	update func(context.Context, service.UpdateRecordRequest) (model.Record, error)
	delete func(context.Context, service.DeleteRecordRequest) error
}

type recordResponseEnvelope struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Payload   json.RawMessage  `json:"payload"`
}

type recordPayloadCase struct {
	name       string
	title      string
	payload    model.RecordPayload
	wantBase64 string
}

var (
	testRecordCreatedAt = time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	testRecordUpdatedAt = time.Date(2026, time.July, 12, 12, 1, 0, 0, time.UTC)
)

func (s recordManagerStub) Create(
	ctx context.Context,
	request service.CreateRecordRequest,
) (model.Record, error) {
	return s.create(ctx, request)
}

func (s recordManagerStub) List(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	return s.list(ctx, userID)
}

func (s recordManagerStub) Get(
	ctx context.Context,
	userID int64,
	recordID string,
) (model.Record, error) {
	return s.get(ctx, userID, recordID)
}

func (s recordManagerStub) Update(
	ctx context.Context,
	request service.UpdateRecordRequest,
) (model.Record, error) {
	return s.update(ctx, request)
}

func (s recordManagerStub) Delete(ctx context.Context, request service.DeleteRecordRequest) error {
	return s.delete(ctx, request)
}

func recordPayloadCases() []recordPayloadCase {
	month := 3
	year := 38

	return []recordPayloadCase{
		{
			name:  "text",
			title: "my note",
			payload: &model.TextPayload{
				Text:     "secret note",
				Metadata: "private metadata",
			},
		},
		{
			name:  "credentials",
			title: "GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://github.com",
				Metadata: "personal account",
			},
		},
		{
			name:  "card",
			title: "Joel's card",
			payload: &model.CardPayload{
				Number:      "2013061420200619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
		{
			name:  "binary",
			title: "Backup",
			payload: &model.BinaryPayload{
				Filename: "backup.bin",
				Data:     []byte{0x00, 0x01, 0x02, 0xff},
				Metadata: "encrypted backup",
			},
			wantBase64: `"data":"AAEC/w=="`,
		},
	}
}

func testRecord(title string, revision int64, payload model.RecordPayload) model.Record {
	return model.Record{
		Metadata: testRecordMetadata(title, payload.RecordType(), revision),
		Payload:  payload,
	}
}

func testRecordMetadata(title string, recordType model.RecordType, revision int64) model.RecordMetadata {
	return model.RecordMetadata{
		ID:        testRecordID,
		Type:      recordType,
		Title:     title,
		Revision:  revision,
		CreatedAt: testRecordCreatedAt,
		UpdatedAt: testRecordUpdatedAt,
	}
}

func serveAuthenticatedRecordHandler(
	t *testing.T,
	handler http.Handler,
	response *httptest.ResponseRecorder,
	request *http.Request,
) {
	t.Helper()

	validator := tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
		if token != "valid-token" {
			t.Fatalf("Validate() token = %q, want valid-token", token)
		}

		return 42, nil
	})
	request.Header.Set("Authorization", "Bearer valid-token")

	middleware.WithAuthentication(handler, validator).ServeHTTP(response, request)
}

func newCreateRecordRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/records", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newUpdateRecordRequest(recordID, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/records/"+recordID, strings.NewReader(body))
	request.SetPathValue("id", recordID)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newDeleteRecordRequest(recordID string) *http.Request {
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/records/"+recordID, nil)
	request.SetPathValue("id", recordID)

	return request
}

func recordRequestBody(t *testing.T, recordType model.RecordType, title string, payload any) string {
	t.Helper()

	body, err := json.Marshal(struct {
		Type    model.RecordType `json:"type"`
		Title   string           `json:"title"`
		Payload any              `json:"payload"`
	}{
		Type:    recordType,
		Title:   title,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("encode record request: %v", err)
	}

	return string(body)
}

func decodeJSONResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func assertCreateRecordRequest(
	t *testing.T,
	got service.CreateRecordRequest,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if got.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", got.UserID)
	}
	if got.Title != wantTitle {
		t.Errorf("Create() title = %q, want %q", got.Title, wantTitle)
	}
	if !reflect.DeepEqual(got.Payload, wantPayload) {
		t.Errorf("Create() payload = %#v, want %#v", got.Payload, wantPayload)
	}
}

func assertUpdateRecordRequest(
	t *testing.T,
	got service.UpdateRecordRequest,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if got.UserID != 42 {
		t.Errorf("Update() userID = %d, want 42", got.UserID)
	}
	if got.RecordID != testRecordID {
		t.Errorf("Update() recordID = %q, want %q", got.RecordID, testRecordID)
	}
	if got.ExpectedRevision != 1 {
		t.Errorf("Update() expected revision = %d, want 1", got.ExpectedRevision)
	}
	if got.Title != wantTitle {
		t.Errorf("Update() title = %q, want %q", got.Title, wantTitle)
	}
	if !reflect.DeepEqual(got.Payload, wantPayload) {
		t.Errorf("Update() payload = %#v, want %#v", got.Payload, wantPayload)
	}
}

func assertRecordResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantRevision int64,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Errorf("status code = %d, want %d", response.Code, wantStatus)
	}
	wantETag := fmt.Sprintf(`"%d"`, wantRevision)
	if got := response.Header().Get("ETag"); got != wantETag {
		t.Errorf("ETag = %q, want %q", got, wantETag)
	}

	var body recordResponseEnvelope
	decodeJSONResponse(t, response, &body)
	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        body.ID,
		Type:      body.Type,
		Title:     body.Title,
		Revision:  body.Revision,
		CreatedAt: body.CreatedAt,
		UpdatedAt: body.UpdatedAt,
	}, newRecordMetadataResponse(testRecordMetadata(wantTitle, wantPayload.RecordType(), wantRevision)))
	assertJSONValue(t, body.Payload, wantPayload)
}

func assertRecordMetadataResponse(t *testing.T, got, want recordMetadataResponse) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("response id = %q, want %q", got.ID, want.ID)
	}
	if got.Type != want.Type {
		t.Errorf("response type = %q, want %q", got.Type, want.Type)
	}
	if got.Title != want.Title {
		t.Errorf("response title = %q, want %q", got.Title, want.Title)
	}
	if got.Revision != want.Revision {
		t.Errorf("response revision = %d, want %d", got.Revision, want.Revision)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("response created_at = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("response updated_at = %s, want %s", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertJSONValue(t *testing.T, gotJSON []byte, want any) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(gotJSON, &gotValue); err != nil {
		t.Fatalf("decode response payload: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("encode expected payload: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatalf("decode expected payload: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("response payload = %#v, want %#v", gotValue, wantValue)
	}
}

func requireBinaryPayload(t *testing.T, payload model.RecordPayload) model.BinaryPayload {
	t.Helper()

	value, ok := payload.(*model.BinaryPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *BinaryPayload", payload)
	}

	return *value
}
