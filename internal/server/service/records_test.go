package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

func TestRecordService_CreateText(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	encrypted := recordcrypto.EncryptedPayload{
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}

	crypto := &recordPayloadCryptoStub{
		encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
			var payload model.TextPayload
			if err := json.Unmarshal(plaintext, &payload); err != nil {
				t.Fatalf("Encrypt() plaintext is not TextPayload JSON: %v", err)
			}
			if payload.Text != "secret note" || payload.Metadata != "private metadata" {
				t.Fatalf("Encrypt() payload = %+v", payload)
			}
			if !strings.Contains(string(aad), "gopherkeeper:v1:user:42:record:") ||
				!strings.Contains(string(aad), ":type:text") {
				t.Fatalf("Encrypt() AAD = %q", aad)
			}

			return encrypted, nil
		},
	}
	records := &recordRepositoryStub{
		createFunc: func(_ context.Context, record model.Record) (model.Record, error) {
			if err := model.ValidateRecordID(record.ID); err != nil {
				t.Fatalf("Create() record ID is invalid: %v", err)
			}
			if record.UserID != 42 || record.Type != model.RecordTypeText || record.Title != "Alice note" {
				t.Fatalf("Create() record = %+v", record)
			}
			if record.Revision != 0 {
				t.Fatalf("Create() revision = %d, want DB default", record.Revision)
			}
			if record.CryptoVersion != encrypted.CryptoVersion || record.KeyID != encrypted.KeyID ||
				!bytes.Equal(record.Nonce, encrypted.Nonce) || !bytes.Equal(record.Ciphertext, encrypted.Ciphertext) {
				t.Fatalf("Create() encrypted record = %+v", record)
			}

			record.Revision = model.RecordInitialRevision
			record.CreatedAt = createdAt
			record.UpdatedAt = updatedAt
			return record, nil
		},
	}
	service := NewRecordService(records, crypto)

	created, err := service.CreateText(context.Background(), CreateTextRecordRequest{
		UserID: 42,
		Title:  "Alice note",
		Payload: model.TextPayload{
			Text:     "secret note",
			Metadata: "private metadata",
		},
	})
	if err != nil {
		t.Fatalf("CreateText() error = %v", err)
	}
	if created.Metadata.Title != "Alice note" || created.Metadata.CreatedAt != createdAt || created.Metadata.UpdatedAt != updatedAt {
		t.Fatalf("CreateText() metadata = %+v", created.Metadata)
	}
	if created.Payload.Text != "secret note" || created.Payload.Metadata != "private metadata" {
		t.Fatalf("CreateText() payload = %+v", created.Payload)
	}
	if crypto.encryptCalls != 1 || records.createCalls != 1 {
		t.Fatalf("calls: Encrypt=%d Create=%d", crypto.encryptCalls, records.createCalls)
	}
}

func TestRecordService_CreateTextValidationError(t *testing.T) {
	crypto := &recordPayloadCryptoStub{}
	records := &recordRepositoryStub{}
	service := NewRecordService(records, crypto)

	_, err := service.CreateText(context.Background(), CreateTextRecordRequest{
		UserID:  42,
		Title:   "Alice note",
		Payload: model.TextPayload{},
	})
	if !errors.Is(err, model.ErrInvalidTextPayload) {
		t.Fatalf("CreateText() error = %v, want ErrInvalidTextPayload", err)
	}
	if crypto.encryptCalls != 0 || records.createCalls != 0 {
		t.Fatalf("calls: Encrypt=%d Create=%d, want 0", crypto.encryptCalls, records.createCalls)
	}
}

func TestRecordService_List(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	want := []model.RecordMetadata{
		{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Type:      model.RecordTypeText,
			Title:     "Alice note",
			Revision:  1,
			UpdatedAt: updatedAt,
		},
	}
	records := &recordRepositoryStub{
		listFunc: func(_ context.Context, userID int64) ([]model.RecordMetadata, error) {
			if userID != 42 {
				t.Fatalf("ListMetadata() userID = %d, want 42", userID)
			}

			return want, nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	got, err := service.List(context.Background(), 42)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_Get(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	textPayload := model.TextPayload{Text: "secret note", Metadata: "private metadata"}
	credentialsPayload := model.CredentialsPayload{
		Login:    "alice",
		Password: "correct-horse-battery-staple",
		URL:      "https://github.com",
		Metadata: "personal account",
	}
	tests := []struct {
		name            string
		recordType      model.RecordType
		payload         any
		wantText        *model.TextPayload
		wantCredentials *model.CredentialsPayload
	}{
		{
			name:       "text record",
			recordType: model.RecordTypeText,
			payload:    textPayload,
			wantText:   &textPayload,
		},
		{
			name:            "credentials record",
			recordType:      model.RecordTypeCredentials,
			payload:         credentialsPayload,
			wantCredentials: &credentialsPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			stored := model.Record{
				ID:            recordID,
				UserID:        42,
				Type:          tt.recordType,
				Title:         "Alice record",
				Revision:      model.RecordInitialRevision,
				CreatedAt:     createdAt,
				UpdatedAt:     updatedAt,
				CryptoVersion: recordcrypto.CryptoVersion,
				KeyID:         recordcrypto.DefaultKeyID,
				Nonce:         []byte("nonce"),
				Ciphertext:    []byte("ciphertext"),
			}
			records := &recordRepositoryStub{
				getFunc: func(_ context.Context, userID int64, gotRecordID string) (model.Record, error) {
					if userID != 42 || gotRecordID != recordID {
						t.Fatalf("Get() args = %d, %q", userID, gotRecordID)
					}

					return stored, nil
				},
			}
			crypto := &recordPayloadCryptoStub{
				decryptFunc: func(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
					if encrypted.CryptoVersion != stored.CryptoVersion || encrypted.KeyID != stored.KeyID ||
						!bytes.Equal(encrypted.Nonce, stored.Nonce) ||
						!bytes.Equal(encrypted.Ciphertext, stored.Ciphertext) {
						t.Fatalf("Decrypt() encrypted payload = %+v", encrypted)
					}
					wantAAD := fmt.Sprintf(
						"gopherkeeper:v1:user:42:record:%s:type:%s",
						recordID,
						tt.recordType,
					)
					if string(aad) != wantAAD {
						t.Fatalf("Decrypt() AAD = %q, want %q", aad, wantAAD)
					}

					return plaintext, nil
				},
			}
			service := NewRecordService(records, crypto)

			got, err := service.Get(context.Background(), 42, recordID)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got.Metadata != stored.Metadata() {
				t.Errorf("Get() metadata = %+v, want %+v", got.Metadata, stored.Metadata())
			}
			switch tt.recordType {
			case model.RecordTypeText:
				payload, ok := textPayloadValue(got.Payload)
				if !ok || tt.wantText == nil || payload != *tt.wantText {
					t.Errorf("Get() text payload = %+v, want %+v", got.Payload, tt.wantText)
				}
			case model.RecordTypeCredentials:
				payload, ok := credentialsPayloadValue(got.Payload)
				if !ok || tt.wantCredentials == nil || payload != *tt.wantCredentials {
					t.Errorf("Get() credentials payload = %+v, want %+v", got.Payload, tt.wantCredentials)
				}
			}
		})
	}
}

func TestRecordService_GetRejectsUnsupportedType(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				UserID: 42,
				Type:   model.RecordTypeBinary,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.Get(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, errInvalidStoredRecord) {
		t.Fatalf("Get() error = %v, want errInvalidStoredRecord", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_GetText(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	stored := model.Record{
		ID:            recordID,
		UserID:        42,
		Type:          model.RecordTypeText,
		Title:         "Alice note",
		Revision:      1,
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}
	payload := model.TextPayload{Text: "secret note", Metadata: "private metadata"}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	records := &recordRepositoryStub{
		getFunc: func(_ context.Context, userID int64, gotRecordID string) (model.Record, error) {
			if userID != 42 || gotRecordID != recordID {
				t.Fatalf("Get() args = %d, %q", userID, gotRecordID)
			}

			return stored, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		decryptFunc: func(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
			if encrypted.CryptoVersion != stored.CryptoVersion || encrypted.KeyID != stored.KeyID ||
				!bytes.Equal(encrypted.Nonce, stored.Nonce) || !bytes.Equal(encrypted.Ciphertext, stored.Ciphertext) {
				t.Fatalf("Decrypt() encrypted payload = %+v", encrypted)
			}
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:text"
			if string(aad) != wantAAD {
				t.Fatalf("Decrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return plaintext, nil
		},
	}
	service := NewRecordService(records, crypto)

	got, err := service.GetText(context.Background(), 42, recordID)
	if err != nil {
		t.Fatalf("GetText() error = %v", err)
	}
	if got.Metadata.ID != recordID || got.Metadata.Title != "Alice note" {
		t.Fatalf("GetText() metadata = %+v", got.Metadata)
	}
	if got.Payload != payload {
		t.Fatalf("GetText() payload = %+v, want %+v", got.Payload, payload)
	}
}

func TestRecordService_GetTextRecordNotFound(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{}, model.ErrRecordNotFound
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.GetText(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("GetText() error = %v, want ErrRecordNotFound", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_GetTextUnsupportedType(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				UserID: 42,
				Type:   model.RecordTypeCredentials,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		decryptFunc: func(recordcrypto.EncryptedPayload, []byte) ([]byte, error) {
			return []byte(`{"login":"alice","password":"secret"}`), nil
		},
	}
	service := NewRecordService(records, crypto)

	_, err := service.GetText(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, model.ErrRecordTypeUnsupported) {
		t.Fatalf("GetText() error = %v, want ErrRecordTypeUnsupported", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_RejectsNilPayload(t *testing.T) {
	var textPayload *model.TextPayload
	var credentialsPayload *model.CredentialsPayload

	tests := []struct {
		name    string
		payload model.RecordPayload
		wantErr error
	}{
		{name: "text", payload: textPayload, wantErr: model.ErrInvalidTextPayload},
		{name: "credentials", payload: credentialsPayload, wantErr: model.ErrInvalidCredentialsPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.Create(context.Background(), CreateRecordRequest{
				UserID:  42,
				Title:   "private record",
				Payload: tt.payload,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}

			_, err = service.Update(context.Background(), UpdateRecordRequest{
				UserID:           42,
				RecordID:         "550e8400-e29b-41d4-a716-446655440000",
				ExpectedRevision: 1,
				Title:            "private record",
				Payload:          tt.payload,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}

			if records.createCalls != 0 || records.getCalls != 0 || records.updateCalls != 0 ||
				crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Create=%d Get=%d Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.createCalls,
					records.getCalls,
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_UpdateText(t *testing.T) {
	fixture := newUpdateTextRecordFixture()
	crypto := newUpdateTextCryptoStub(t, fixture)
	records := newUpdateTextRepositoryStub(t, fixture)
	service := NewRecordService(records, crypto)

	updated, err := service.UpdateText(context.Background(), UpdateTextRecordRequest{
		UserID:           42,
		RecordID:         fixture.recordID,
		ExpectedRevision: 1,
		Title:            "Updated Alice note",
		Payload:          fixture.payload,
	})
	if err != nil {
		t.Fatalf("UpdateText() error = %v", err)
	}

	assertUpdatedTextRecord(t, updated, fixture)
	assertRecordServiceCalls(t, records, crypto, recordServiceCallCounts{
		get:     1,
		update:  1,
		encrypt: 1,
	})
}

type updateTextRecordFixture struct {
	recordID  string
	createdAt time.Time
	updatedAt time.Time
	current   model.Record
	encrypted recordcrypto.EncryptedPayload
	payload   model.TextPayload
}

func newUpdateTextRecordFixture() updateTextRecordFixture {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 9, 12, 5, 0, 0, time.UTC)

	return updateTextRecordFixture{
		recordID:  recordID,
		createdAt: createdAt,
		updatedAt: updatedAt,
		current: model.Record{
			ID:            recordID,
			UserID:        42,
			Type:          model.RecordTypeText,
			Title:         "Alice note",
			Revision:      1,
			CreatedAt:     createdAt,
			UpdatedAt:     createdAt,
			CryptoVersion: recordcrypto.CryptoVersion,
			KeyID:         recordcrypto.DefaultKeyID,
			Nonce:         []byte("old nonce"),
			Ciphertext:    []byte("old ciphertext"),
		},
		encrypted: recordcrypto.EncryptedPayload{
			CryptoVersion: recordcrypto.CryptoVersion,
			KeyID:         recordcrypto.DefaultKeyID,
			Nonce:         []byte("new nonce"),
			Ciphertext:    []byte("new ciphertext"),
		},
		payload: model.TextPayload{Text: "updated secret note", Metadata: "updated private metadata"},
	}
}

func newUpdateTextCryptoStub(t *testing.T, fixture updateTextRecordFixture) *recordPayloadCryptoStub {
	t.Helper()

	return &recordPayloadCryptoStub{
		encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
			assertEncryptedTextPayload(t, plaintext, fixture.payload)
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:text"
			if string(aad) != wantAAD {
				t.Fatalf("Encrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return fixture.encrypted, nil
		},
	}
}

func assertEncryptedTextPayload(t *testing.T, plaintext []byte, want model.TextPayload) {
	t.Helper()

	var got model.TextPayload
	if err := json.Unmarshal(plaintext, &got); err != nil {
		t.Fatalf("Encrypt() plaintext is not TextPayload JSON: %v", err)
	}
	if got != want {
		t.Fatalf("Encrypt() payload = %+v, want %+v", got, want)
	}
}

func newUpdateTextRepositoryStub(t *testing.T, fixture updateTextRecordFixture) *recordRepositoryStub {
	t.Helper()

	return &recordRepositoryStub{
		getFunc: func(_ context.Context, userID int64, recordID string) (model.Record, error) {
			assertUpdateTextGetArgs(t, userID, recordID, fixture)
			return fixture.current, nil
		},
		updateFunc: func(_ context.Context, record model.Record, expectedRevision int64) (model.Record, error) {
			assertUpdateTextRecordPatch(t, record, expectedRevision, fixture)
			updated := record
			updated.Revision = 2
			updated.CreatedAt = fixture.createdAt
			updated.UpdatedAt = fixture.updatedAt
			return updated, nil
		},
	}
}

func assertUpdateTextGetArgs(t *testing.T, userID int64, recordID string, fixture updateTextRecordFixture) {
	t.Helper()

	if userID != 42 || recordID != fixture.recordID {
		t.Fatalf("Get() args = %d, %q", userID, recordID)
	}
}

func assertUpdateTextRecordPatch(
	t *testing.T,
	record model.Record,
	expectedRevision int64,
	fixture updateTextRecordFixture,
) {
	t.Helper()

	if expectedRevision != 1 {
		t.Fatalf("Update() expectedRevision = %d, want 1", expectedRevision)
	}
	if record.ID != fixture.recordID || record.UserID != 42 || record.Type != model.RecordTypeText {
		t.Fatalf("Update() record identity = %+v", record)
	}
	if record.Title != "Updated Alice note" {
		t.Fatalf("Update() title = %q, want Updated Alice note", record.Title)
	}
	if bytes.Equal(record.Nonce, fixture.current.Nonce) || bytes.Equal(record.Ciphertext, fixture.current.Ciphertext) {
		t.Fatalf("Update() reused old encrypted payload: %+v", record)
	}
	if record.CryptoVersion != fixture.encrypted.CryptoVersion || record.KeyID != fixture.encrypted.KeyID ||
		!bytes.Equal(record.Nonce, fixture.encrypted.Nonce) ||
		!bytes.Equal(record.Ciphertext, fixture.encrypted.Ciphertext) {
		t.Fatalf("Update() encrypted record = %+v", record)
	}
}

func assertUpdatedTextRecord(t *testing.T, got TextRecord, fixture updateTextRecordFixture) {
	t.Helper()

	if got.Metadata.ID != fixture.recordID || got.Metadata.Title != "Updated Alice note" ||
		got.Metadata.Revision != 2 || got.Metadata.CreatedAt != fixture.createdAt ||
		got.Metadata.UpdatedAt != fixture.updatedAt {
		t.Fatalf("UpdateText() metadata = %+v", got.Metadata)
	}
	if got.Payload != fixture.payload {
		t.Fatalf("UpdateText() payload = %+v, want %+v", got.Payload, fixture.payload)
	}
}

type recordServiceCallCounts struct {
	get     int
	update  int
	encrypt int
	decrypt int
}

func assertRecordServiceCalls(
	t *testing.T,
	records *recordRepositoryStub,
	crypto *recordPayloadCryptoStub,
	want recordServiceCallCounts,
) {
	t.Helper()

	if records.getCalls != want.get || records.updateCalls != want.update ||
		crypto.encryptCalls != want.encrypt || crypto.decryptCalls != want.decrypt {
		t.Fatalf(
			"calls: Get=%d Update=%d Encrypt=%d Decrypt=%d",
			records.getCalls,
			records.updateCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}

func TestRecordService_UpdateTextValidationError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		request UpdateTextRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: UpdateTextRecordRequest{
				UserID:           0,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          model.TextPayload{Text: "secret note"},
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid record ID",
			request: UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         "not-a-uuid",
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 0,
				Title:            "Alice note",
				Payload:          model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordRevision,
		},
		{
			name: "invalid title",
			request: UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "   ",
				Payload:          model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name: "invalid payload",
			request: UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          model.TextPayload{},
			},
			wantErr: model.ErrInvalidTextPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := &recordPayloadCryptoStub{}
			records := &recordRepositoryStub{}
			service := NewRecordService(records, crypto)

			_, err := service.UpdateText(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateText() error = %v, want %v", err, tt.wantErr)
			}
			if records.getCalls != 0 || records.updateCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Get=%d Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.getCalls,
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_UpdateTextCurrentRecordErrors(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		current model.Record
		getErr  error
		wantErr error
	}{
		{
			name:    "record not found",
			getErr:  model.ErrRecordNotFound,
			wantErr: model.ErrRecordNotFound,
		},
		{
			name: "unsupported type",
			current: model.Record{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeCredentials,
				Revision: 1,
			},
			wantErr: model.ErrRecordTypeUnsupported,
		},
		{
			name: "stale revision",
			current: model.Record{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeText,
				Revision: 2,
			},
			wantErr: model.ErrRecordRevisionConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				getFunc: func(context.Context, int64, string) (model.Record, error) {
					return tt.current, tt.getErr
				},
			}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.UpdateText(context.Background(), UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          model.TextPayload{Text: "secret note"},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateText() error = %v, want %v", err, tt.wantErr)
			}
			if records.updateCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_UpdateTextCryptoError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errCrypto := errors.New("crypto unavailable")
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeText,
				Revision: 1,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		encryptFunc: func([]byte, []byte) (recordcrypto.EncryptedPayload, error) {
			return recordcrypto.EncryptedPayload{}, errCrypto
		},
	}
	service := NewRecordService(records, crypto)

	_, err := service.UpdateText(context.Background(), UpdateTextRecordRequest{
		UserID:           42,
		RecordID:         recordID,
		ExpectedRevision: 1,
		Title:            "Alice note",
		Payload:          model.TextPayload{Text: "secret note"},
	})
	if !errors.Is(err, errCrypto) {
		t.Fatalf("UpdateText() error = %v, want crypto error", err)
	}
	if records.updateCalls != 0 || crypto.encryptCalls != 1 || crypto.decryptCalls != 0 {
		t.Fatalf(
			"calls: Update=%d Encrypt=%d Decrypt=%d",
			records.updateCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}

func TestRecordService_UpdateTextRepositoryError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errRepository := errors.New("repository unavailable")
	tests := []struct {
		name      string
		updateErr error
		wantErr   error
	}{
		{
			name:      "revision conflict",
			updateErr: model.ErrRecordRevisionConflict,
			wantErr:   model.ErrRecordRevisionConflict,
		},
		{
			name:      "record not found",
			updateErr: model.ErrRecordNotFound,
			wantErr:   model.ErrRecordNotFound,
		},
		{
			name:      "repository error",
			updateErr: errRepository,
			wantErr:   errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				getFunc: func(context.Context, int64, string) (model.Record, error) {
					return model.Record{
						ID:       recordID,
						UserID:   42,
						Type:     model.RecordTypeText,
						Revision: 1,
					}, nil
				},
				updateFunc: func(context.Context, model.Record, int64) (model.Record, error) {
					return model.Record{}, tt.updateErr
				},
			}
			crypto := &recordPayloadCryptoStub{
				encryptFunc: func([]byte, []byte) (recordcrypto.EncryptedPayload, error) {
					return recordcrypto.EncryptedPayload{
						CryptoVersion: recordcrypto.CryptoVersion,
						KeyID:         recordcrypto.DefaultKeyID,
						Nonce:         []byte("nonce"),
						Ciphertext:    []byte("ciphertext"),
					}, nil
				},
			}
			service := NewRecordService(records, crypto)

			_, err := service.UpdateText(context.Background(), UpdateTextRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          model.TextPayload{Text: "secret note"},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateText() error = %v, want %v", err, tt.wantErr)
			}
			if records.updateCalls != 1 || crypto.encryptCalls != 1 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Update=%d Encrypt=%d Decrypt=%d",
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_Delete(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	records := &recordRepositoryStub{
		deleteFunc: func(_ context.Context, userID int64, gotRecordID string, expectedRevision int64) error {
			if userID != 42 || gotRecordID != recordID || expectedRevision != 1 {
				t.Fatalf("Delete() args = %d, %q, %d", userID, gotRecordID, expectedRevision)
			}

			return nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	if err := service.Delete(context.Background(), DeleteRecordRequest{
		UserID:           42,
		RecordID:         recordID,
		ExpectedRevision: 1,
	}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if records.getCalls != 0 || records.deleteCalls != 1 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
		t.Fatalf(
			"calls: Get=%d Delete=%d Encrypt=%d Decrypt=%d",
			records.getCalls,
			records.deleteCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}

func TestRecordService_DeleteValidationError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		request DeleteRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: DeleteRecordRequest{
				UserID:           0,
				RecordID:         recordID,
				ExpectedRevision: 1,
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid record ID",
			request: DeleteRecordRequest{
				UserID:           42,
				RecordID:         "not-a-uuid",
				ExpectedRevision: 1,
			},
			wantErr: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: DeleteRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 0,
			},
			wantErr: model.ErrInvalidRecordRevision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			err := service.Delete(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if records.deleteCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Delete=%d Encrypt=%d Decrypt=%d, want 0",
					records.deleteCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_DeleteRepositoryError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errRepository := errors.New("repository unavailable")
	tests := []struct {
		name      string
		deleteErr error
		wantErr   error
	}{
		{
			name:      "revision conflict",
			deleteErr: model.ErrRecordRevisionConflict,
			wantErr:   model.ErrRecordRevisionConflict,
		},
		{
			name:      "record not found",
			deleteErr: model.ErrRecordNotFound,
			wantErr:   model.ErrRecordNotFound,
		},
		{
			name:      "repository error",
			deleteErr: errRepository,
			wantErr:   errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				deleteFunc: func(context.Context, int64, string, int64) error {
					return tt.deleteErr
				},
			}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			err := service.Delete(context.Background(), DeleteRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if records.getCalls != 0 || records.deleteCalls != 1 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Get=%d Delete=%d Encrypt=%d Decrypt=%d",
					records.getCalls,
					records.deleteCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

type recordRepositoryStub struct {
	createFunc func(context.Context, model.Record) (model.Record, error)
	listFunc   func(context.Context, int64) ([]model.RecordMetadata, error)
	getFunc    func(context.Context, int64, string) (model.Record, error)
	updateFunc func(context.Context, model.Record, int64) (model.Record, error)
	deleteFunc func(context.Context, int64, string, int64) error

	createCalls int
	listCalls   int
	getCalls    int
	updateCalls int
	deleteCalls int
}

func (s *recordRepositoryStub) Create(ctx context.Context, record model.Record) (model.Record, error) {
	s.createCalls++
	if s.createFunc == nil {
		return model.Record{}, errors.New("unexpected Create call")
	}

	return s.createFunc(ctx, record)
}

func (s *recordRepositoryStub) ListMetadata(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	s.listCalls++
	if s.listFunc == nil {
		return nil, errors.New("unexpected ListMetadata call")
	}

	return s.listFunc(ctx, userID)
}

func (s *recordRepositoryStub) Get(ctx context.Context, userID int64, recordID string) (model.Record, error) {
	s.getCalls++
	if s.getFunc == nil {
		return model.Record{}, errors.New("unexpected Get call")
	}

	return s.getFunc(ctx, userID, recordID)
}

func (s *recordRepositoryStub) Update(
	ctx context.Context,
	record model.Record,
	expectedRevision int64,
) (model.Record, error) {
	s.updateCalls++
	if s.updateFunc == nil {
		return model.Record{}, errors.New("unexpected Update call")
	}

	return s.updateFunc(ctx, record, expectedRevision)
}

func (s *recordRepositoryStub) Delete(
	ctx context.Context,
	userID int64,
	recordID string,
	expectedRevision int64,
) error {
	s.deleteCalls++
	if s.deleteFunc == nil {
		return errors.New("unexpected Delete call")
	}

	return s.deleteFunc(ctx, userID, recordID, expectedRevision)
}

type recordPayloadCryptoStub struct {
	encryptFunc func([]byte, []byte) (recordcrypto.EncryptedPayload, error)
	decryptFunc func(recordcrypto.EncryptedPayload, []byte) ([]byte, error)

	encryptCalls int
	decryptCalls int
}

func (s *recordPayloadCryptoStub) Encrypt(plaintext []byte, aad []byte) (recordcrypto.EncryptedPayload, error) {
	s.encryptCalls++
	if s.encryptFunc == nil {
		return recordcrypto.EncryptedPayload{}, errors.New("unexpected Encrypt call")
	}

	return s.encryptFunc(plaintext, aad)
}

func (s *recordPayloadCryptoStub) Decrypt(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
	s.decryptCalls++
	if s.decryptFunc == nil {
		return nil, errors.New("unexpected Decrypt call")
	}

	return s.decryptFunc(encrypted, aad)
}
