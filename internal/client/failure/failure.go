// Package failure классифицирует ошибки клиентского runtime без привязки UI к тексту ошибок.
package failure

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/url"
	"strings"
	"syscall"
)

// Kind описывает категорию ошибки, значимую для пользовательских интерфейсов Клиента.
type Kind int

const (
	// Unknown обозначает ошибку, для которой не удалось определить категорию.
	Unknown Kind = iota

	// Canceled обозначает отменённую операцию.
	Canceled

	// Unavailable обозначает недоступный Сервер или отказ в соединении.
	Unavailable

	// HostNotFound обозначает ошибку разрешения имени узла.
	HostNotFound

	// NetworkUnreachable обозначает недоступную сеть или маршрут.
	NetworkUnreachable

	// Timeout обозначает истечение времени ожидания операции.
	Timeout

	// TLSCertificate обозначает ошибку проверки TLS-сертификата.
	TLSCertificate

	// TLSHandshake обозначает ошибку TLS-handshake.
	TLSHandshake

	// HTTPSRequired обозначает попытку HTTPS-подключения к HTTP endpoint.
	HTTPSRequired

	// Unauthorized обозначает отсутствие действующей авторизации.
	Unauthorized

	// Conflict обозначает конфликт конкурентного изменения.
	Conflict

	// NotFound обозначает отсутствие запрошенного ресурса.
	NotFound

	// Validation обозначает некорректные пользовательские данные.
	Validation

	// TooLarge обозначает превышение допустимого размера данных.
	TooLarge
)

// Error хранит типизированную клиентскую ошибку, диагностический контекст
// и безопасное сообщение для пользовательского интерфейса.
type Error struct {
	kind        Kind
	detail      string
	userMessage string
	cause       error
}

// New создаёт типизированную клиентскую ошибку с одинаковым диагностическим
// и пользовательским сообщением.
func New(kind Kind, message string, cause error) error {
	return Wrap(kind, message, message, cause)
}

// Wrap создаёт типизированную ошибку, сохраняя отдельные диагностический
// контекст и безопасное пользовательское сообщение.
func Wrap(kind Kind, detail, userMessage string, cause error) error {
	return &Error{
		kind:        kind,
		detail:      strings.TrimSpace(detail),
		userMessage: strings.TrimSpace(userMessage),
		cause:       cause,
	}
}

// Error возвращает диагностическое описание с исходной причиной.
func (e *Error) Error() string {
	switch {
	case e.detail != "" && e.cause != nil:
		return e.detail + ": " + e.cause.Error()
	case e.detail != "":
		return e.detail
	case e.cause != nil:
		return e.cause.Error()
	default:
		return "client error"
	}
}

// Unwrap возвращает исходную причину ошибки.
func (e *Error) Unwrap() error { return e.cause }

// FailureKind возвращает категорию ошибки.
func (e *Error) FailureKind() Kind { return e.kind }

// UserMessage возвращает безопасное сообщение для пользователя.
func (e *Error) UserMessage() string {
	if e.userMessage != "" {
		return e.userMessage
	}
	return Reason(e.kind)
}

type kindProvider interface{ FailureKind() Kind }
type messageProvider interface{ UserMessage() string }

// KindOf определяет категорию ошибки по типизированным обёрткам и системным причинам.
func KindOf(err error) Kind {
	if err == nil {
		return Unknown
	}

	var provider kindProvider
	if errors.As(err, &provider) {
		return provider.FailureKind()
	}
	if kind := kindFromKnownErrors(err); kind != Unknown {
		return kind
	}
	if kind := kindFromTypedNetworkErrors(err); kind != Unknown {
		return kind
	}

	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		if kind := KindOf(urlError.Err); kind != Unknown {
			return kind
		}
	}

	return kindFromErrorMessage(err.Error())
}

func kindFromKnownErrors(err error) Kind {
	switch {
	case errors.Is(err, context.Canceled):
		return Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return Timeout
	case errors.Is(err, syscall.ECONNREFUSED):
		return Unavailable
	case errors.Is(err, syscall.ENETUNREACH), errors.Is(err, syscall.EHOSTUNREACH):
		return NetworkUnreachable
	default:
		return Unknown
	}
}

func kindFromTypedNetworkErrors(err error) Kind {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		if dnsError.IsTimeout {
			return Timeout
		}
		return HostNotFound
	}
	var certificateAuthority x509.UnknownAuthorityError
	if errors.As(err, &certificateAuthority) {
		return TLSCertificate
	}
	var hostnameError x509.HostnameError
	if errors.As(err, &hostnameError) {
		return TLSCertificate
	}
	var certificateInvalid x509.CertificateInvalidError
	if errors.As(err, &certificateInvalid) {
		return TLSCertificate
	}
	var recordHeaderError tls.RecordHeaderError
	if errors.As(err, &recordHeaderError) {
		return TLSHandshake
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return Timeout
	}

	return Unknown
}

func kindFromErrorMessage(value string) Kind {
	// Некоторые ошибки net/http не имеют публичного отдельного типа.
	message := strings.ToLower(value)
	switch {
	case strings.Contains(message, "server gave http response to https client"):
		return HTTPSRequired
	case strings.Contains(message, "x509") || strings.Contains(message, "certificate"):
		return TLSCertificate
	case strings.Contains(message, "tls"):
		return TLSHandshake
	case strings.Contains(message, "no such host") || strings.Contains(message, "name or service not known"):
		return HostNotFound
	case strings.Contains(message, "network is unreachable"):
		return NetworkUnreachable
	case strings.Contains(message, "connection refused"):
		return Unavailable
	case strings.Contains(message, "timeout") || strings.Contains(message, "deadline exceeded"):
		return Timeout
	default:
		return Unknown
	}
}

// Message возвращает наиболее близкое безопасное пользовательское сообщение.
func Message(err error) string {
	if err == nil {
		return ""
	}
	var provider messageProvider
	if errors.As(err, &provider) {
		return strings.TrimSpace(provider.UserMessage())
	}
	return strings.TrimSpace(err.Error())
}

// Context добавляет диагностический контекст, не включая его в сообщение для пользователя.
func Context(operation string, err error) error {
	if err == nil {
		return nil
	}

	operation = strings.TrimSpace(operation)
	if operation == "" {
		return err
	}

	return Wrap(KindOf(err), operation, Message(err), err)
}

// Network преобразует сетевую ошибку в типизированную форму, сохраняя
// диагностическое имя операции.
func Network(operation string, err error) error {
	kind := KindOf(err)
	if kind == Unknown {
		return Wrap(Unknown, operation, "Connection failed", err)
	}
	return Wrap(kind, operation, Reason(kind), err)
}

// Reason возвращает стабильное пользовательское описание категории.
func Reason(kind Kind) string {
	switch kind {
	case Canceled:
		return "Operation canceled"
	case Unavailable:
		return "Connection refused"
	case HostNotFound:
		return "Host not found"
	case NetworkUnreachable:
		return "Network unreachable"
	case Timeout:
		return "Connection timed out"
	case TLSCertificate:
		return "Certificate verification failed"
	case TLSHandshake:
		return "TLS handshake failed"
	case HTTPSRequired:
		return "Server does not support HTTPS"
	case Unauthorized:
		return "Not authorized"
	case Conflict:
		return "Conflict"
	case NotFound:
		return "Not found"
	case Validation:
		return "Invalid data"
	case TooLarge:
		return "Payload exceeds the allowed size"
	default:
		return "Connection failed"
	}
}
