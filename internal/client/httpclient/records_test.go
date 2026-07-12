package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestClient_CreateRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	password := "correct-horse-battery-staple"
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleCreateRecordTestRequest(t, createdAt, password, w, r)
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record, err := client.CreateRecord(
		context.Background(),
		"test.jwt.token",
		"GitHub",
		&model.CredentialsPayload{
			Login:    "alice",
			Password: password,
			URL:      "https://github.com",
		},
	)
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	assertCreatedCredentialsRecord(t, record, password)
}

func handleCreateRecordTestRequest(
	t *testing.T,
	createdAt time.Time,
	password string,
	w http.ResponseWriter,
	r *http.Request,
) {
	t.Helper()

	if r.Method != http.MethodPost {
		t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
	}
	if r.URL.Path != recordsPath {
		t.Errorf("path = %s, want %s", r.URL.Path, recordsPath)
	}
	if strings.Contains(r.URL.RawQuery, password) || strings.Contains(r.Header.Get("Authorization"), password) {
		t.Error("credentials password appeared outside JSON body")
	}

	request := decodeCredentialsRecordRequest(t, r)
	if request.Payload.Password != password {
		t.Error("credentials password was not transferred unchanged")
	}
	writeRecordResponse(t, w, http.StatusCreated, recordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeCredentials,
		Title:     request.Title,
		Revision:  1,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, &request.Payload)
}

type credentialsRecordRequest struct {
	Title   string
	Payload model.CredentialsPayload
}

func decodeCredentialsRecordRequest(t *testing.T, r *http.Request) credentialsRecordRequest {
	t.Helper()

	var request struct {
		Type    model.RecordType `json:"type"`
		Title   string           `json:"title"`
		Payload json.RawMessage  `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	if request.Type != model.RecordTypeCredentials {
		t.Errorf("type = %q, want credentials", request.Type)
	}

	var payload model.CredentialsPayload
	if err := json.Unmarshal(request.Payload, &payload); err != nil {
		t.Fatalf("decode credentials payload: %v", err)
	}

	return credentialsRecordRequest{Title: request.Title, Payload: payload}
}

func assertCreatedCredentialsRecord(t *testing.T, record model.Record, password string) {
	t.Helper()

	payload, ok := record.Payload.(*model.CredentialsPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *model.CredentialsPayload", record.Payload)
	}
	if payload.Login != "alice" || payload.Password != password {
		t.Errorf("payload = %#v, want original credentials", payload)
	}
	if record.Metadata.Revision != 1 {
		t.Errorf("revision = %d, want 1", record.Metadata.Revision)
	}
}

func TestClient_CreateBinaryRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	data := []byte{0x00, 0x01, 0x02, 0xff}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}

		var request struct {
			Type    model.RecordType `json:"type"`
			Title   string           `json:"title"`
			Payload struct {
				Filename    string `json:"filename"`
				Data        string `json:"data"`
				ContentType string `json:"content_type"`
				Metadata    string `json:"metadata"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Type != model.RecordTypeBinary {
			t.Errorf("type = %q, want binary", request.Type)
		}
		if request.Payload.Data != "AAEC/w==" {
			t.Errorf("data = %q, want Base64-encoded bytes", request.Payload.Data)
		}

		payload := &model.BinaryPayload{
			Filename:    request.Payload.Filename,
			Data:        data,
			ContentType: request.Payload.ContentType,
			Metadata:    request.Payload.Metadata,
		}
		writeRecordResponse(t, w, http.StatusCreated, recordResponse{
			ID:        testRecordID,
			Type:      model.RecordTypeBinary,
			Title:     request.Title,
			Revision:  1,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		}, payload)
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record, err := client.CreateRecord(
		context.Background(),
		"test.jwt.token",
		"Alice encrypted backup",
		&model.BinaryPayload{
			Filename:    "backup.bin",
			Data:        data,
			ContentType: "application/octet-stream",
			Metadata:    "private backup",
		},
	)
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}

	payload, ok := record.Payload.(*model.BinaryPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *model.BinaryPayload", record.Payload)
	}
	if !bytes.Equal(payload.Data, data) {
		t.Errorf("data = %v, want %v", payload.Data, data)
	}
}

type clientGetRecordTestCase struct {
	name       string
	recordType model.RecordType
	payload    model.RecordPayload
}

func TestClient_GetRecord(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	tests := []clientGetRecordTestCase{
		{
			name:       "text",
			recordType: model.RecordTypeText,
			payload:    &model.TextPayload{Text: "secret note", Metadata: "private"},
		},
		{
			name:       "credentials",
			recordType: model.RecordTypeCredentials,
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
			},
		},
		{
			name:       "card",
			recordType: model.RecordTypeCard,
			payload: &model.CardPayload{
				Number:      "2013 0614 2020 0619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
		{
			name:       "binary",
			recordType: model.RecordTypeBinary,
			payload: &model.BinaryPayload{
				Filename:    "backup.bin",
				Data:        []byte{0x00, 0x01, 0x02, 0xff},
				ContentType: "application/octet-stream",
				Metadata:    "private backup",
			},
		},
		{
			name:       "empty binary",
			recordType: model.RecordTypeBinary,
			payload: &model.BinaryPayload{
				Filename: "empty.bin",
				Data:     []byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClientGetRecord(t, tt)
		})
	}
}

func testClientGetRecord(t *testing.T, tt clientGetRecordTestCase) {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleGetRecordTestRequest(t, tt, w, r)
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record, err := client.GetRecord(context.Background(), "test.jwt.token", testRecordID)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if record.Metadata.Type != tt.recordType {
		t.Errorf("type = %q, want %q", record.Metadata.Type, tt.recordType)
	}
	assertClientRecordPayloadEqual(t, record.Payload, tt.payload)
}

func handleGetRecordTestRequest(
	t *testing.T,
	tt clientGetRecordTestCase,
	w http.ResponseWriter,
	r *http.Request,
) {
	t.Helper()

	if r.Method != http.MethodGet {
		t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
	}
	if r.URL.Path != recordsPath+"/"+testRecordID {
		t.Errorf("path = %s, want record path", r.URL.Path)
	}

	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	writeRecordResponse(t, w, http.StatusOK, recordResponse{
		ID:        testRecordID,
		Type:      tt.recordType,
		Title:     "Record",
		Revision:  1,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, tt.payload)
}

func assertClientRecordPayloadEqual(t *testing.T, got, want model.RecordPayload) {
	t.Helper()

	switch want := want.(type) {
	case *model.TextPayload:
		payload, ok := got.(*model.TextPayload)
		if !ok || *payload != *want {
			t.Fatalf("payload = %#v, want text payload %#v", got, want)
		}
	case *model.CredentialsPayload:
		payload, ok := got.(*model.CredentialsPayload)
		if !ok || *payload != *want {
			t.Fatalf("payload = %#v, want credentials payload %#v", got, want)
		}
	case *model.CardPayload:
		payload, ok := got.(*model.CardPayload)
		if !ok || !reflect.DeepEqual(payload, want) {
			t.Fatalf("payload = %#v, want card payload %#v", got, want)
		}
	case *model.BinaryPayload:
		payload, ok := got.(*model.BinaryPayload)
		if !ok || !reflect.DeepEqual(payload, want) {
			t.Fatalf("payload = %#v, want binary payload %#v", got, want)
		}
	default:
		t.Fatalf("unsupported expected payload type %T", want)
	}
}

func TestClient_UpdateRecord(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 10, 12, 5, 0, 0, time.UTC)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPut)
		}
		if got := r.Header.Get("If-Match"); got != `"1"` {
			t.Errorf("If-Match = %q, want quoted revision", got)
		}

		var request struct {
			Type    model.RecordType `json:"type"`
			Title   string           `json:"title"`
			Payload json.RawMessage  `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Type != model.RecordTypeCredentials {
			t.Errorf("type = %q, want credentials", request.Type)
		}

		var payload model.CredentialsPayload
		if err := json.Unmarshal(request.Payload, &payload); err != nil {
			t.Fatalf("decode credentials: %v", err)
		}

		writeRecordResponse(t, w, http.StatusOK, recordResponse{
			ID:        testRecordID,
			Type:      model.RecordTypeCredentials,
			Title:     request.Title,
			Revision:  2,
			CreatedAt: time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC),
			UpdatedAt: updatedAt,
		}, &payload)
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record, err := client.UpdateRecord(
		context.Background(),
		"test.jwt.token",
		testRecordID,
		1,
		"Updated GitHub",
		&model.CredentialsPayload{
			Login:    "alice",
			Password: "updated-correct-horse-battery-staple",
		},
	)
	if err != nil {
		t.Fatalf("UpdateRecord() error = %v", err)
	}
	if record.Metadata.Revision != 2 {
		t.Errorf("revision = %d, want 2", record.Metadata.Revision)
	}
	if !record.Metadata.UpdatedAt.Equal(updatedAt) {
		t.Errorf("updated at = %s, want %s", record.Metadata.UpdatedAt, updatedAt)
	}
}

func TestClient_UpdateRecordReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"code":"record_revision_conflict","message":"record revision conflict"}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.UpdateRecord(context.Background(), "test.jwt.token", testRecordID, 1, "Updated note", &model.TextPayload{Text: "updated secret"})
	if err == nil {
		t.Fatal("UpdateRecord() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("UpdateRecord() error = %T, want *APIError", err)
	}
	if apiError.Code != "record_revision_conflict" {
		t.Errorf("code = %q, want record_revision_conflict", apiError.Code)
	}
	if strings.Contains(err.Error(), "test.jwt.token") || strings.Contains(err.Error(), "updated secret") {
		t.Error("update error contains access token or payload")
	}
}

func TestClient_GetRecordRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"550e8400-e29b-41d4-a716-446655440000",
			"type":"credentials",
			"title":"GitHub",
			"revision":1,
			"created_at":"2026-07-10T12:00:00Z",
			"updated_at":"2026-07-10T12:00:00Z",
			"payload":{"login":"alice","password":"correct-horse-battery-staple","text":"unexpected"}
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.GetRecord(context.Background(), "test.jwt.token", testRecordID)
	if err == nil {
		t.Fatal("GetRecord() error = nil, want invalid payload error")
	}
	if strings.Contains(err.Error(), "correct-horse-battery-staple") {
		t.Error("decode error contains credentials password")
	}
}

func TestClient_GetRecordRejectsInvalidBinaryPayload(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"550e8400-e29b-41d4-a716-446655440000",
			"type":"binary",
			"title":"Alice encrypted backup",
			"revision":1,
			"created_at":"2026-07-12T12:00:00Z",
			"updated_at":"2026-07-12T12:00:00Z",
			"payload":{"filename":"backup.bin","data":"not-base64***"}
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.GetRecord(context.Background(), "test.jwt.token", testRecordID)
	if err == nil {
		t.Fatal("GetRecord() error = nil, want invalid binary payload error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") || strings.Contains(err.Error(), "not-base64") {
		t.Error("decode error contains access token or binary data")
	}
}

func TestClient_DeleteRecord(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want %s", r.Method, http.MethodDelete)
		}
		if got := r.Header.Get("If-Match"); got != `"2"` {
			t.Errorf("If-Match = %q, want quoted revision", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := client.DeleteRecord(context.Background(), "test.jwt.token", testRecordID, 2); err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
}

func writeRecordResponse(
	t *testing.T,
	w http.ResponseWriter,
	status int,
	response recordResponse,
	payload model.RecordPayload,
) {
	t.Helper()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal response payload: %v", err)
	}
	response.Payload = payloadJSON

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}
