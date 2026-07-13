package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	mebibyte = 1024 * 1024

	// TextPayloadMaxSize содержит максимальный размер text payload в байтах UTF-8.
	TextPayloadMaxSize = mebibyte

	// BinaryPayloadMaxSize содержит максимальный размер binary payload в исходных байтах.
	BinaryPayloadMaxSize = 2 * mebibyte

	// MetadataMaxSize содержит максимальный размер приватной метаинформации в байтах UTF-8.
	MetadataMaxSize = 64 * 1024

	// HTTPRequestBodyMaxSize содержит максимальный размер HTTP request body в байтах.
	HTTPRequestBodyMaxSize int64 = 4 * mebibyte

	// RecordInitialRevision содержит начальную ревизию новой записи.
	RecordInitialRevision int64 = 1
)

var (
	// ErrRecordNotFound сообщает, что приватная запись не найдена или недоступна пользователю.
	ErrRecordNotFound = errors.New("record not found")

	// ErrRecordTypeUnsupported сообщает, что тип приватной записи не поддерживается.
	ErrRecordTypeUnsupported = errors.New("record type unsupported")

	// ErrPayloadTooLarge сообщает, что приватный payload превышает допустимый размер.
	ErrPayloadTooLarge = errors.New("payload too large")

	// ErrInvalidRecordID сообщает, что идентификатор записи не является UUID.
	ErrInvalidRecordID = errors.New("invalid record id")

	// ErrInvalidRecordTitle сообщает, что открытое название записи некорректно.
	ErrInvalidRecordTitle = errors.New("invalid record title")

	// ErrInvalidTextPayload сообщает, что text payload некорректен.
	ErrInvalidTextPayload = errors.New("invalid text payload")

	// ErrInvalidBinaryPayload сообщает, что binary payload некорректен.
	ErrInvalidBinaryPayload = errors.New("invalid binary payload")

	// ErrRecordRevisionConflict сообщает, что ожидаемая ревизия записи устарела.
	ErrRecordRevisionConflict = errors.New("record revision conflict")

	// ErrRecordPreconditionRequired сообщает, что операция над записью требует ожидаемую ревизию.
	ErrRecordPreconditionRequired = errors.New("record precondition required")

	// ErrInvalidRecordRevision сообщает, что ревизия записи некорректна.
	ErrInvalidRecordRevision = errors.New("invalid record revision")

	// ErrInvalidRecordData сообщает, что данные приватной записи некорректны.
	ErrInvalidRecordData = errors.New("invalid record data")
)

// RecordType описывает тип приватной записи.
type RecordType string

const (
	// RecordTypeCredentials обозначает запись с парой login/password.
	RecordTypeCredentials RecordType = "credentials"

	// RecordTypeCard обозначает запись с данными банковской карты.
	RecordTypeCard RecordType = "card"

	// RecordTypeText обозначает запись с произвольным текстом.
	RecordTypeText RecordType = "text"

	// RecordTypeBinary обозначает запись с произвольными бинарными данными.
	RecordTypeBinary RecordType = "binary"
)

// Validate проверяет, что тип записи поддерживается доменной моделью.
func (recordType RecordType) Validate() error {
	switch recordType {
	case RecordTypeCredentials, RecordTypeCard, RecordTypeText, RecordTypeBinary:
		return nil
	default:
		return ErrRecordTypeUnsupported
	}
}

// Record содержит открытые поля и типизированный приватный payload записи.
type Record struct {
	// Metadata содержит открытые поля приватной записи.
	Metadata RecordMetadata

	// Payload содержит типизированный приватный payload.
	Payload RecordPayload
}

// EncryptedRecord описывает зашифрованную приватную запись в серверном хранилище.
type EncryptedRecord struct {
	// ID содержит UUID приватной записи.
	ID string

	// UserID содержит идентификатор владельца записи.
	UserID int64

	// Type содержит неизменяемый тип payload записи.
	Type RecordType

	// Title содержит открытое название записи.
	Title string

	// Revision содержит текущую ревизию записи для оптимистической блокировки.
	Revision int64

	// CreatedAt содержит время создания записи в UTC.
	CreatedAt time.Time

	// UpdatedAt содержит время последнего изменения записи в UTC.
	UpdatedAt time.Time

	// CryptoVersion содержит версию формата серверного шифрования payload.
	CryptoVersion int

	// KeyID содержит идентификатор мастер-ключа, которым зашифрован payload.
	KeyID string

	// Nonce содержит уникальный nonce AES-GCM.
	Nonce []byte

	// Ciphertext содержит зашифрованный приватный payload вместе с authentication tag.
	Ciphertext []byte
}

// RecordMetadata содержит открытые поля приватной записи без payload.
type RecordMetadata struct {
	// ID содержит UUID приватной записи.
	ID string

	// Type содержит тип payload записи.
	Type RecordType

	// Title содержит открытое название записи.
	Title string

	// Revision содержит текущую ревизию записи.
	Revision int64

	// CreatedAt содержит время создания записи в UTC.
	CreatedAt time.Time

	// UpdatedAt содержит время последнего изменения записи в UTC.
	UpdatedAt time.Time
}

// Metadata возвращает открытые поля записи без encrypted payload.
func (record EncryptedRecord) Metadata() RecordMetadata {
	return RecordMetadata{
		ID:        record.ID,
		Type:      record.Type,
		Title:     record.Title,
		Revision:  record.Revision,
		CreatedAt: record.CreatedAt,
		UpdatedAt: record.UpdatedAt,
	}
}

// ValidateRecordTitle проверяет открытое название приватной записи.
func ValidateRecordTitle(title string) error {
	if strings.TrimSpace(title) == "" || !utf8.ValidString(title) {
		return ErrInvalidRecordTitle
	}

	return nil
}

// NewRecordID создаёт новый UUID v4 для приватной записи.
func NewRecordID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate record id: %w", err)
	}

	return id.String(), nil
}

// ValidateRecordID проверяет, что идентификатор записи является UUID.
func ValidateRecordID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return ErrInvalidRecordID
	}

	return nil
}

// ValidateRecordRevision проверяет, что ревизия записи является положительным числом.
func ValidateRecordRevision(revision int64) error {
	if revision <= 0 {
		return ErrInvalidRecordRevision
	}

	return nil
}
