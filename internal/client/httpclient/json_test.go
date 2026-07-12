package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestClient_DoJSONReturnsStatusErrorForInvalidErrorResponse(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"code":`},
		{name: "missing code", body: `{"message":"internal server error"}`},
		{name: "missing message", body: `{"code":"internal_error"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := New(serverAddress(server), writeServerCertificate(t, server))
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			err = client.doJSON(context.Background(), jsonRequest{
				operation:      "test operation",
				method:         http.MethodGet,
				path:           "/",
				accessToken:    "test.jwt.token",
				expectedStatus: http.StatusOK,
			})
			if err == nil {
				t.Fatal("doJSON() error = nil, want status error")
			}
			if !strings.Contains(err.Error(), "500 Internal Server Error") {
				t.Errorf("doJSON() error = %q, want status 500", err)
			}
			if strings.Contains(err.Error(), tt.body) || strings.Contains(err.Error(), "test.jwt.token") {
				t.Errorf("doJSON() error contains response body or access token: %q", err)
			}
		})
	}
}

func TestClient_DoJSONReturnsDecodeError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"value":`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var response struct {
		Value string `json:"value"`
	}
	err = client.doJSON(context.Background(), jsonRequest{
		operation:      "test operation",
		method:         http.MethodPost,
		path:           "/",
		accessToken:    "test.jwt.token",
		requestBody:    map[string]string{"secret": "private-payload"},
		expectedStatus: http.StatusOK,
		responseBody:   &response,
	})
	if err == nil {
		t.Fatal("doJSON() error = nil, want decode error")
	}
	if !strings.Contains(err.Error(), "decode test operation response") {
		t.Errorf("doJSON() error = %q, want decode context", err)
	}
	if strings.Contains(err.Error(), "test.jwt.token") || strings.Contains(err.Error(), "private-payload") {
		t.Errorf("doJSON() error contains access token or request payload: %q", err)
	}
}

func TestClient_DoJSONReturnsNetworkError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	certificate := writeServerCertificate(t, server)
	address := serverAddress(server)
	server.Close()

	client, err := New(address, certificate)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err = client.doJSON(context.Background(), jsonRequest{
		operation:      "test operation",
		method:         http.MethodPost,
		path:           "/",
		accessToken:    "test.jwt.token",
		requestBody:    map[string]string{"secret": "private-payload"},
		expectedStatus: http.StatusOK,
	})
	if err == nil {
		t.Fatal("doJSON() error = nil, want network error")
	}
	if !strings.Contains(err.Error(), "send test operation request") {
		t.Errorf("doJSON() error = %q, want send context", err)
	}
	if strings.Contains(err.Error(), "test.jwt.token") || strings.Contains(err.Error(), "private-payload") {
		t.Errorf("doJSON() error contains access token or request payload: %q", err)
	}
}

func TestDecodeAPIErrorMapsSemanticCause(t *testing.T) {
	tests := []struct {
		code string
		want error
	}{
		{code: "login_already_exists", want: model.ErrLoginAlreadyExists},
		{code: "invalid_credentials", want: model.ErrInvalidCredentials},
		{code: "unauthorized", want: model.ErrUnauthorized},
		{code: "record_not_found", want: model.ErrRecordNotFound},
		{code: "record_revision_conflict", want: model.ErrRecordRevisionConflict},
		{code: "precondition_required", want: model.ErrRecordPreconditionRequired},
		{code: "payload_too_large", want: model.ErrPayloadTooLarge},
		{code: "invalid_request", want: model.ErrInvalidRecordData},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := decodeAPIError(400, "400 Bad Request", []byte(
				`{"code":"`+tt.code+`","message":"safe message"}`,
			))
			if !errors.Is(err, tt.want) {
				t.Fatalf("decodeAPIError() error = %v, want %v", err, tt.want)
			}
		})
	}
}
