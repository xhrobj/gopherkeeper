package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestClient_UpdateTextRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 9, 12, 5, 0, 0, time.UTC)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertUpdateTextRecordRequest(t, r) {
			http.Error(w, "invalid update request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"2"`)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(textRecordResponse{
			ID:        testRecordID,
			Type:      model.RecordTypeText,
			Title:     "Updated note",
			Revision:  2,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Payload: model.TextPayload{
				Text:     "updated secret",
				Metadata: "updated private metadata",
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record, err := client.UpdateTextRecord(context.Background(), "test.jwt.token", testRecordID, 1, UpdateTextRecordRequest{
		Title: "Updated note",
		Payload: model.TextPayload{
			Text:     "updated secret",
			Metadata: "updated private metadata",
		},
	})
	if err != nil {
		t.Fatalf("UpdateTextRecord() error = %v", err)
	}

	if record.Metadata.ID != testRecordID {
		t.Errorf("record ID = %q, want %q", record.Metadata.ID, testRecordID)
	}
	if record.Metadata.Revision != 2 {
		t.Errorf("revision = %d, want 2", record.Metadata.Revision)
	}
	if record.Payload.Text != "updated secret" {
		t.Errorf("text = %q, want updated secret", record.Payload.Text)
	}
	if !record.Metadata.UpdatedAt.Equal(updatedAt) {
		t.Errorf("updated at = %s, want %s", record.Metadata.UpdatedAt, updatedAt)
	}
}

func assertUpdateTextRecordRequest(t *testing.T, r *http.Request) bool {
	t.Helper()

	valid := true
	if r.Method != http.MethodPut {
		t.Errorf("method = %s, want %s", r.Method, http.MethodPut)
		valid = false
	}
	if r.URL.Path != recordsPath+"/"+testRecordID {
		t.Errorf("path = %s, want %s", r.URL.Path, recordsPath+"/"+testRecordID)
		valid = false
	}
	if got := r.Header.Get("Authorization"); got != "Bearer test.jwt.token" {
		t.Errorf("Authorization = %q, want bearer token", got)
		valid = false
	}
	if got := r.Header.Get("If-Match"); got != `"1"` {
		t.Errorf("If-Match = %q, want quoted revision", got)
		valid = false
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
		valid = false
	}

	var request updateRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Errorf("decode request: %v", err)
		return false
	}
	if request.Type != model.RecordTypeText {
		t.Errorf("type = %q, want text", request.Type)
		valid = false
	}
	if request.Title != "Updated note" {
		t.Errorf("title = %q, want Updated note", request.Title)
		valid = false
	}
	if request.Payload.Text != "updated secret" {
		t.Error("text payload was not transferred unchanged")
		valid = false
	}
	if request.Payload.Metadata != "updated private metadata" {
		t.Error("metadata was not transferred unchanged")
		valid = false
	}

	return valid
}

func TestClient_UpdateTextRecordReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{
			"code":"record_revision_conflict",
			"message":"record revision conflict"
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.UpdateTextRecord(context.Background(), "test.jwt.token", testRecordID, 1, UpdateTextRecordRequest{
		Title:   "Updated note",
		Payload: model.TextPayload{Text: "updated secret"},
	})
	if err == nil {
		t.Fatal("UpdateTextRecord() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("UpdateTextRecord() error = %T, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusConflict {
		t.Errorf("status code = %d, want %d", apiError.StatusCode, http.StatusConflict)
	}
	if apiError.Code != "record_revision_conflict" {
		t.Errorf("code = %q, want record_revision_conflict", apiError.Code)
	}
	if strings.Contains(err.Error(), "test.jwt.token") || strings.Contains(err.Error(), "updated secret") {
		t.Error("update error contains access token or text payload")
	}
}

func TestClient_DeleteRecord(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want %s", r.Method, http.MethodDelete)
		}
		if r.URL.Path != recordsPath+"/"+testRecordID {
			t.Errorf("path = %s, want %s", r.URL.Path, recordsPath+"/"+testRecordID)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test.jwt.token" {
			t.Errorf("Authorization = %q, want bearer token", got)
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

func TestClient_DeleteRecordReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{
			"code":"record_not_found",
			"message":"record not found"
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.DeleteRecord(context.Background(), "test.jwt.token", testRecordID, 2)
	if err == nil {
		t.Fatal("DeleteRecord() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("DeleteRecord() error = %T, want *APIError", err)
	}
	if apiError.StatusCode != http.StatusNotFound {
		t.Errorf("status code = %d, want %d", apiError.StatusCode, http.StatusNotFound)
	}
	if apiError.Code != "record_not_found" {
		t.Errorf("code = %q, want record_not_found", apiError.Code)
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("delete error contains access token")
	}
}
