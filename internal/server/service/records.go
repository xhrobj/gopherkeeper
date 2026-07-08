package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

var errInvalidRecordOwner = errors.New("invalid record owner")

// RecordCreator сохраняет encrypted record.
type RecordCreator interface {
	// Create сохраняет encrypted record и возвращает зафиксированное состояние.
	Create(ctx context.Context, record model.Record) (model.Record, error)
}

// RecordReader читает encrypted records и metadata пользователя.
type RecordReader interface {
	// ListMetadata возвращает открытые поля записей пользователя без payload.
	ListMetadata(ctx context.Context, userID int64) ([]model.RecordMetadata, error)

	// Get возвращает encrypted record пользователя по идентификатору.
	Get(ctx context.Context, userID int64, recordID string) (model.Record, error)
}

// RecordPayloadCrypto шифрует и расшифровывает приватные payload'ы записей.
type RecordPayloadCrypto interface {
	// Encrypt шифрует plaintext payload с authenticated data.
	Encrypt(plaintext []byte, aad []byte) (recordcrypto.EncryptedPayload, error)

	// Decrypt расшифровывает encrypted payload с authenticated data.
	Decrypt(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error)
}

// RecordRepository объединяет операции хранения, необходимые RecordService.
type RecordRepository interface {
	RecordCreator
	RecordReader
}

// CreateTextRecordRequest содержит входные данные для создания text-записи.
type CreateTextRecordRequest struct {
	UserID  int64
	Title   string
	Payload model.TextPayload
}

// TextRecord содержит открытую metadata и расшифрованный text payload.
type TextRecord struct {
	Metadata model.RecordMetadata
	Payload  model.TextPayload
}

// RecordService реализует серверные сценарии приватных записей.
type RecordService struct {
	records RecordRepository
	crypto  RecordPayloadCrypto
}

// NewRecordService создаёт сервис приватных записей.
func NewRecordService(records RecordRepository, crypto RecordPayloadCrypto) *RecordService {
	return &RecordService{
		records: records,
		crypto:  crypto,
	}
}

// CreateText создаёт text-запись, шифрует payload и сохраняет encrypted record.
func (s *RecordService) CreateText(ctx context.Context, request CreateTextRecordRequest) (TextRecord, error) {
	if err := validateRecordOwner(request.UserID); err != nil {
		return TextRecord{}, err
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return TextRecord{}, err
	}
	if err := request.Payload.Validate(); err != nil {
		return TextRecord{}, err
	}

	recordID, err := model.NewRecordID()
	if err != nil {
		return TextRecord{}, err
	}

	plaintext, err := json.Marshal(request.Payload)
	if err != nil {
		return TextRecord{}, fmt.Errorf("marshal text payload: %w", err)
	}

	aad, err := recordcrypto.BuildAAD(request.UserID, recordID, model.RecordTypeText)
	if err != nil {
		return TextRecord{}, fmt.Errorf("build record AAD: %w", err)
	}

	encrypted, err := s.crypto.Encrypt(plaintext, aad)
	if err != nil {
		return TextRecord{}, fmt.Errorf("encrypt text payload: %w", err)
	}

	record, err := s.records.Create(ctx, model.Record{
		ID:            recordID,
		UserID:        request.UserID,
		Type:          model.RecordTypeText,
		Title:         request.Title,
		CryptoVersion: encrypted.CryptoVersion,
		KeyID:         encrypted.KeyID,
		Nonce:         encrypted.Nonce,
		Ciphertext:    encrypted.Ciphertext,
	})
	if err != nil {
		return TextRecord{}, fmt.Errorf("create text record: %w", err)
	}

	return TextRecord{
		Metadata: record.Metadata(),
		Payload:  request.Payload,
	}, nil
}

// List возвращает открытые поля записей пользователя без расшифрования payload'ов.
func (s *RecordService) List(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	if err := validateRecordOwner(userID); err != nil {
		return nil, err
	}

	metadata, err := s.records.ListMetadata(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	return metadata, nil
}

// GetText возвращает text-запись пользователя и расшифровывает её payload.
func (s *RecordService) GetText(ctx context.Context, userID int64, recordID string) (TextRecord, error) {
	if err := validateRecordOwner(userID); err != nil {
		return TextRecord{}, err
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return TextRecord{}, err
	}

	record, err := s.records.Get(ctx, userID, recordID)
	if err != nil {
		return TextRecord{}, fmt.Errorf("get text record: %w", err)
	}
	if record.Type != model.RecordTypeText {
		return TextRecord{}, model.ErrRecordTypeUnsupported
	}

	aad, err := recordcrypto.BuildAAD(record.UserID, record.ID, record.Type)
	if err != nil {
		return TextRecord{}, fmt.Errorf("build record AAD: %w", err)
	}

	plaintext, err := s.crypto.Decrypt(recordcrypto.EncryptedPayload{
		CryptoVersion: record.CryptoVersion,
		KeyID:         record.KeyID,
		Nonce:         record.Nonce,
		Ciphertext:    record.Ciphertext,
	}, aad)
	if err != nil {
		return TextRecord{}, fmt.Errorf("decrypt text payload: %w", err)
	}

	var payload model.TextPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return TextRecord{}, fmt.Errorf("unmarshal text payload: %w", err)
	}
	if err := payload.Validate(); err != nil {
		return TextRecord{}, err
	}

	return TextRecord{
		Metadata: record.Metadata(),
		Payload:  payload,
	}, nil
}

func validateRecordOwner(userID int64) error {
	if userID <= 0 {
		return errInvalidRecordOwner
	}

	return nil
}
