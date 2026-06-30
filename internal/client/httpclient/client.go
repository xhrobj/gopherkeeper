package httpclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type healthResponse struct {
	Status string `json:"status"`
}

func New(address, caCertFile string) (*Client, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system CA certificates: %w", err)
	}

	if caCertFile != "" {
		certificate, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("read additional CA certificate: %w", err)
		}

		if ok := rootCAs.AppendCertsFromPEM(certificate); !ok {
			return nil, errors.New("parse additional CA certificate")
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}

	return &Client{
		baseURL: "https://" + address,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   time.Second * 10,
		},
	}, nil
}

func (c *Client) Health(ctx context.Context) (string, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/health",
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("create health request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("send health request: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("health request returned status %s", response.Status)
	}

	var health healthResponse
	if err := json.NewDecoder(response.Body).Decode(&health); err != nil {
		return "", fmt.Errorf("decode health response: %w", err)
	}

	return health.Status, nil
}
