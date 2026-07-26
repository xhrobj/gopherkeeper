package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/client/transporttls"
)

const requestTimeout = 10 * time.Second

type idleConnectionCloser interface {
	CloseIdleConnections()
}

// Client выполняет HTTPS-запросы к Серверу GophKeeper.
type Client struct {
	client          *resty.Client
	idleConnections idleConnectionCloser
}

type healthResponse struct {
	Status string `json:"status"`
}

// New создаёт HTTPS-Клиент с системными корневыми сертификатами
// и дополнительным доверенным CA certificate при его наличии.
func New(address, caCertFile string) (*Client, error) {
	tlsConfig, err := transporttls.NewConfig(caCertFile)
	if err != nil {
		return nil, err
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = tlsConfig

	restyClient := resty.NewWithClient(&http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	})
	restyClient.SetBaseURL("https://" + address)

	return &Client{client: restyClient, idleConnections: transport}, nil
}

// Close закрывает неиспользуемые соединения HTTPS-транспорта, принадлежащего Клиенту.
func (c *Client) Close() error {
	if c == nil || c.idleConnections == nil {
		return nil
	}

	c.idleConnections.CloseIdleConnections()

	return nil
}

// Health проверяет доступность Сервера и возвращает его технический статус.
func (c *Client) Health(ctx context.Context) (string, error) {
	response, err := c.client.R().
		SetContext(ctx).
		Get("/health")
	if err != nil {
		return "", healthRequestError(err)
	}

	if response.StatusCode() != http.StatusOK {
		return "", failure.Wrap(
			healthStatusFailureKind(response.StatusCode()),
			fmt.Sprintf("health request returned status %s", response.Status()),
			"Server health check failed",
			nil,
		)
	}

	var health healthResponse
	if err := json.Unmarshal(response.Body(), &health); err != nil {
		return "", failure.Wrap(
			failure.Unknown,
			"decode health response",
			"Invalid server health response",
			err,
		)
	}

	return health.Status, nil
}

func healthStatusFailureKind(statusCode int) failure.Kind {
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return failure.Timeout
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return failure.Unavailable
	default:
		return failure.Unknown
	}
}

func healthRequestError(err error) error {
	operation := "send health request"

	if failure.KindOf(err) == failure.Unavailable {
		operation = "server unavailable"
	}

	return failure.Network(operation, err)
}
