package transporttls

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

// NewConfig создаёт TLS-конфигурацию с системными корневыми сертификатами
// и дополнительным доверенным CA certificate при его наличии.
func NewConfig(caCertFile string) (*tls.Config, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, failure.Wrap(
			failure.TLSCertificate,
			"load system CA certificates",
			"Unable to load system CA certificates",
			err,
		)
	}

	if caCertFile != "" {
		certificate, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, failure.Wrap(
				failure.TLSCertificate,
				"read additional CA certificate",
				"Unable to read additional CA certificate",
				err,
			)
		}

		if !rootCAs.AppendCertsFromPEM(certificate) {
			return nil, failure.Wrap(
				failure.TLSCertificate,
				"parse additional CA certificate",
				"Unable to parse additional CA certificate",
				nil,
			)
		}
	}

	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    rootCAs,
	}, nil
}
