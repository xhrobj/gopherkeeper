package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestBinaryPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload *BinaryPayload
		wantErr error
	}{
		{
			name: "full payload",
			payload: &BinaryPayload{
				Filename:    "alice-secret.bin",
				Data:        []byte{0x00, 0x2a, 0xff},
				ContentType: "application/octet-stream",
				Metadata:    "private binary",
			},
		},
		{
			name: "empty file",
			payload: &BinaryPayload{
				Filename: "empty.bin",
				Data:     []byte{},
			},
		},
		{
			name: "data at limit",
			payload: &BinaryPayload{
				Filename: "limit.bin",
				Data:     bytes.Repeat([]byte{0x2a}, BinaryPayloadMaxSize),
			},
		},
		{
			name: "metadata at limit",
			payload: &BinaryPayload{
				Filename: "metadata.bin",
				Data:     []byte{0x2a},
				Metadata: strings.Repeat("a", MetadataMaxSize),
			},
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "empty filename",
			payload: &BinaryPayload{
				Data: []byte{0x2a},
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "blank filename",
			payload: &BinaryPayload{
				Filename: " \t\n",
				Data:     []byte{0x2a},
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "missing data",
			payload: &BinaryPayload{
				Filename: "missing.bin",
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "data too large",
			payload: &BinaryPayload{
				Filename: "large.bin",
				Data:     bytes.Repeat([]byte{0x2a}, BinaryPayloadMaxSize+1),
			},
			wantErr: ErrPayloadTooLarge,
		},
		{
			name: "invalid UTF-8 filename",
			payload: &BinaryPayload{
				Filename: string([]byte{0xff}),
				Data:     []byte{0x2a},
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "invalid UTF-8 content type",
			payload: &BinaryPayload{
				Filename:    "secret.bin",
				Data:        []byte{0x2a},
				ContentType: string([]byte{0xff}),
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "invalid UTF-8 metadata",
			payload: &BinaryPayload{
				Filename: "secret.bin",
				Data:     []byte{0x2a},
				Metadata: string([]byte{0xff}),
			},
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "metadata too large",
			payload: &BinaryPayload{
				Filename: "secret.bin",
				Data:     []byte{0x2a},
				Metadata: strings.Repeat("a", MetadataMaxSize+1),
			},
			wantErr: ErrPayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBinaryPayload_JSONDataPresence(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantNil bool
		wantErr error
	}{
		{
			name:    "missing data",
			body:    `{"filename":"empty.bin"}`,
			wantNil: true,
			wantErr: ErrInvalidBinaryPayload,
		},
		{
			name: "empty file",
			body: `{"filename":"empty.bin","data":""}`,
		},
		{
			name:    "null data",
			body:    `{"filename":"empty.bin","data":null}`,
			wantNil: true,
			wantErr: ErrInvalidBinaryPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var payload BinaryPayload
			if err := json.Unmarshal([]byte(tt.body), &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if (payload.Data == nil) != tt.wantNil {
				t.Fatalf("Data nil = %t, want %t", payload.Data == nil, tt.wantNil)
			}
			if err := payload.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestBinaryPayload_ValidatePreservesValues(t *testing.T) {
	payload := BinaryPayload{
		Filename:    " secret.bin ",
		Data:        []byte{0x00, 0x2a, 0xff},
		ContentType: " application/octet-stream ",
		Metadata:    " private binary ",
	}
	want := BinaryPayload{
		Filename:    payload.Filename,
		Data:        append([]byte(nil), payload.Data...),
		ContentType: payload.ContentType,
		Metadata:    payload.Metadata,
	}

	if err := payload.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("Validate() payload = %+v, want %+v", payload, want)
	}
}
