package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type requestPayload struct {
	Value string `json:"value"`
}

func TestDecodeJSONValue(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    requestPayload
		wantErr bool
		isErr   error
	}{
		{
			name: "valid JSON",
			body: `{"value":"secret"}`,
			want: requestPayload{Value: "secret"},
		},
		{
			name:    "empty body",
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			body:    `{"value":"secret"`,
			wantErr: true,
		},
		{
			name:    "unknown field",
			body:    `{"value":"secret","extra":42}`,
			wantErr: true,
		},
		{
			name:    "multiple JSON values",
			body:    `{"value":"first"}{}`,
			wantErr: true,
			isErr:   errMultipleJSONValues,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got requestPayload
			err := decodeJSONValue(strings.NewReader(tt.body), &got)

			if tt.wantErr {
				if err == nil {
					t.Fatal("decodeJSONValue() error = nil")
				}
				if tt.isErr != nil && !errors.Is(err, tt.isErr) {
					t.Fatalf("decodeJSONValue() error = %v, want errors.Is(%v)", err, tt.isErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("decodeJSONValue() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("decodeJSONValue() payload = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestDecodeJSONPayload(t *testing.T) {
	var got requestPayload
	if err := decodeJSONPayload(json.RawMessage(`{"value":"secret"}`), &got); err != nil {
		t.Fatalf("decodeJSONPayload() error = %v", err)
	}
	if want := (requestPayload{Value: "secret"}); got != want {
		t.Fatalf("decodeJSONPayload() payload = %#v, want %#v", got, want)
	}
}

func TestDecodeJSONRequest_EnforcesBodyLimit(t *testing.T) {
	body := `{"value":"` + strings.Repeat("a", int(maxRequestBodySize)) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	response := httptest.NewRecorder()
	var target requestPayload

	err := decodeJSONRequest(response, request, &target)
	if !isRequestBodyTooLarge(err) {
		t.Fatalf("decodeJSONRequest() error = %v, want body too large", err)
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "JSON", value: "application/json", want: true},
		{name: "JSON with charset", value: "application/json; charset=utf-8", want: true},
		{name: "missing", value: ""},
		{name: "plain text", value: "text/plain"},
		{name: "malformed", value: "application/json; charset"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJSONContentType(tt.value); got != tt.want {
				t.Fatalf("isJSONContentType(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}
