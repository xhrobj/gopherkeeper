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
	// TextPayloadMaxSize содержит максимальный размер text payload в байтах UTF-8.
	TextPayloadMaxSize = 1 * 1024 * 1024

	// MetadataMaxSize содержит максимальный размер приватной метаинформации в байтах UTF-8.
	MetadataMaxSize = 64 * 1024

	// HTTPRequestBodyMaxSize содержит максимальный размер HTTP request body в байтах.
	HTTPRequestBodyMaxSize int64 = 4 * 1024 * 1024

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

	// ErrRecordRevisionConflict сообщает, что ожидаемая ревизия записи устарела.
	ErrRecordRevisionConflict = errors.New("record revision conflict")

	// ErrRecordPreconditionRequired сообщает, что операция над записью требует ожидаемую ревизию.
	ErrRecordPreconditionRequired = errors.New("record precondition required")

	// ErrInvalidRecordRevision сообщает, что ревизия записи некорректна.
	ErrInvalidRecordRevision = errors.New("invalid record revision")
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

// Record описывает приватную запись в серверном хранилище.
type Record struct {
	ID            string
	UserID        int64
	Type          RecordType
	Title         string
	Revision      int64
	CreatedAt     time.Time
	UpdatedAt     time.Time
	CryptoVersion int
	KeyID         string
	Nonce         []byte
	Ciphertext    []byte
}

// RecordMetadata содержит открытые поля приватной записи без payload.
type RecordMetadata struct {
	ID        string
	Type      RecordType
	Title     string
	Revision  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TextPayload содержит приватный текстовый payload записи.
type TextPayload struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательный текст и ограничения размера text payload.
func (payload TextPayload) Validate() error {
	if payload.Text == "" || !utf8.ValidString(payload.Text) || !utf8.ValidString(payload.Metadata) {
		return ErrInvalidTextPayload
	}

	if len(payload.Text) > TextPayloadMaxSize || len(payload.Metadata) > MetadataMaxSize {
		return ErrPayloadTooLarge
	}

	return nil
}

// Metadata возвращает открытые поля записи без encrypted payload.
func (record Record) Metadata() RecordMetadata {
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
