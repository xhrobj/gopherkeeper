package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestDecodeAPIError_DoesNotAssignSemanticCause(t *testing.T) {
	err := decodeAPIError(
		http.StatusBadRequest,
		"400 Bad Request",
		[]byte(`{"code":"invalid_request","message":"safe message"}`),
	)

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("decodeAPIError() error = %T, want *APIError", err)
	}
	if apiError.cause != nil {
		t.Errorf("decodeAPIError() cause = %v, want nil", apiError.cause)
	}
}
