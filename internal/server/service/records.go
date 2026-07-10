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

// RecordUpdater изменяет encrypted record при совпадении ожидаемой ревизии.
type RecordUpdater interface {
	// Update изменяет encrypted record и возвращает зафиксированное состояние.
	Update(ctx context.Context, record model.Record, expectedRevision int64) (model.Record, error)
}

// RecordDeleter удаляет encrypted record при совпадении ожидаемой ревизии.
type RecordDeleter interface {
	// Delete физически удаляет encrypted record.
	Delete(ctx context.Context, userID int64, recordID string, expectedRevision int64) error
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
	RecordUpdater
	RecordDeleter
}

// CreateTextRecordRequest содержит входные данные для создания text-записи.
type CreateTextRecordRequest struct {
	UserID  int64
	Title   string
	Payload model.TextPayload
}

// UpdateTextRecordRequest содержит входные данные для изменения text-записи.
type UpdateTextRecordRequest struct {
	UserID           int64
	RecordID         string
	ExpectedRevision int64
	Title            string
	Payload          model.TextPayload
}

// DeleteRecordRequest содержит входные данные для удаления приватной записи.
type DeleteRecordRequest struct {
	UserID           int64
	RecordID         string
	ExpectedRevision int64
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
	record, err := s.createRecord(
		ctx,
		request.UserID,
		request.Title,
		model.RecordTypeText,
		request.Payload,
	)
	if err != nil {
		return TextRecord{}, err
	}

	return TextRecord{
		Metadata: record.Metadata(),
		Payload:  request.Payload,
	}, nil
}

// UpdateText изменяет text-запись при совпадении ожидаемой ревизии.
func (s *RecordService) UpdateText(ctx context.Context, request UpdateTextRecordRequest) (TextRecord, error) {
	updated, err := s.updateRecord(
		ctx,
		request.UserID,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		model.RecordTypeText,
		request.Payload,
	)
	if err != nil {
		return TextRecord{}, err
	}

	return TextRecord{
		Metadata: updated.Metadata(),
		Payload:  request.Payload,
	}, nil
}

// Delete удаляет приватную запись при совпадении ожидаемой ревизии.
func (s *RecordService) Delete(ctx context.Context, request DeleteRecordRequest) error {
	if err := validateRecordOwner(request.UserID); err != nil {
		return err
	}
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return err
	}

	if err := s.records.Delete(ctx, request.UserID, request.RecordID, request.ExpectedRevision); err != nil {
		return fmt.Errorf("delete record: %w", err)
	}

	return nil
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
	var payload model.TextPayload

	record, err := s.getRecord(ctx, userID, recordID, model.RecordTypeText, &payload)
	if err != nil {
		return TextRecord{}, err
	}

	return TextRecord{
		Metadata: record.Metadata(),
		Payload:  payload,
	}, nil
}

func (s *RecordService) createRecord(
	ctx context.Context,
	userID int64,
	title string,
	recordType model.RecordType,
	payload recordPayload,
) (model.Record, error) {
	if err := validateRecordOwner(userID); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordTitle(title); err != nil {
		return model.Record{}, err
	}
	if err := recordType.Validate(); err != nil {
		return model.Record{}, err
	}
	if err := payload.Validate(); err != nil {
		return model.Record{}, err
	}

	recordID, err := model.NewRecordID()
	if err != nil {
		return model.Record{}, err
	}

	encrypted, err := s.encryptRecordPayload(userID, recordID, recordType, payload)
	if err != nil {
		return model.Record{}, err
	}

	record, err := s.records.Create(ctx, model.Record{
		ID:            recordID,
		UserID:        userID,
		Type:          recordType,
		Title:         title,
		CryptoVersion: encrypted.CryptoVersion,
		KeyID:         encrypted.KeyID,
		Nonce:         encrypted.Nonce,
		Ciphertext:    encrypted.Ciphertext,
	})
	if err != nil {
		return model.Record{}, fmt.Errorf("create %s record: %w", recordType, err)
	}
	return record, nil
}

func (s *RecordService) updateRecord(
	ctx context.Context,
	userID int64,
	recordID string,
	expectedRevision int64,
	title string,
	recordType model.RecordType,
	payload recordPayload,
) (model.Record, error) {
	if err := validateRecordOwner(userID); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordRevision(expectedRevision); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordTitle(title); err != nil {
		return model.Record{}, err
	}
	if err := recordType.Validate(); err != nil {
		return model.Record{}, err
	}
	if err := payload.Validate(); err != nil {
		return model.Record{}, err
	}

	current, err := s.records.Get(ctx, userID, recordID)
	if err != nil {
		return model.Record{}, fmt.Errorf("get %s record for update: %w", recordType, err)
	}
	if current.Type != recordType {
		return model.Record{}, model.ErrRecordTypeUnsupported
	}
	if current.Revision != expectedRevision {
		return model.Record{}, model.ErrRecordRevisionConflict
	}

	encrypted, err := s.encryptRecordPayload(userID, recordID, recordType, payload)
	if err != nil {
		return model.Record{}, err
	}

	updated, err := s.records.Update(ctx, model.Record{
		ID:            recordID,
		UserID:        userID,
		Type:          recordType,
		Title:         title,
		CryptoVersion: encrypted.CryptoVersion,
		KeyID:         encrypted.KeyID,
		Nonce:         encrypted.Nonce,
		Ciphertext:    encrypted.Ciphertext,
	}, expectedRevision)
	if err != nil {
		return model.Record{}, fmt.Errorf("update %s record: %w", recordType, err)
	}
	if updated.Type != recordType {
		return model.Record{}, model.ErrRecordTypeUnsupported
	}

	return updated, nil
}

func (s *RecordService) getRecord(
	ctx context.Context,
	userID int64,
	recordID string,
	recordType model.RecordType,
	payload recordPayload,
) (model.Record, error) {
	if err := validateRecordOwner(userID); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return model.Record{}, err
	}
	if err := recordType.Validate(); err != nil {
		return model.Record{}, err
	}

	record, err := s.records.Get(ctx, userID, recordID)
	if err != nil {
		return model.Record{}, fmt.Errorf("get %s record: %w", recordType, err)
	}
	if record.Type != recordType {
		return model.Record{}, model.ErrRecordTypeUnsupported
	}

	if err := s.decryptRecordPayload(record, payload); err != nil {
		return model.Record{}, err
	}

	return record, nil
}

func (s *RecordService) encryptRecordPayload(
	userID int64,
	recordID string,
	recordType model.RecordType,
	payload recordPayload,
) (recordcrypto.EncryptedPayload, error) {
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return recordcrypto.EncryptedPayload{}, fmt.Errorf("marshal %s payload: %w", recordType, err)
	}

	aad, err := recordcrypto.BuildAAD(userID, recordID, recordType)
	if err != nil {
		return recordcrypto.EncryptedPayload{}, fmt.Errorf("build record AAD: %w", err)
	}

	encrypted, err := s.crypto.Encrypt(plaintext, aad)
	if err != nil {
		return recordcrypto.EncryptedPayload{}, fmt.Errorf("encrypt %s payload: %w", recordType, err)
	}

	return encrypted, nil
}

func (s *RecordService) decryptRecordPayload(record model.Record, payload recordPayload) error {
	aad, err := recordcrypto.BuildAAD(record.UserID, record.ID, record.Type)
	if err != nil {
		return fmt.Errorf("build record AAD: %w", err)
	}

	plaintext, err := s.crypto.Decrypt(recordcrypto.EncryptedPayload{
		CryptoVersion: record.CryptoVersion,
		KeyID:         record.KeyID,
		Nonce:         record.Nonce,
		Ciphertext:    record.Ciphertext,
	}, aad)
	if err != nil {
		return fmt.Errorf("decrypt %s payload: %w", record.Type, err)
	}

	if err := json.Unmarshal(plaintext, payload); err != nil {
		return fmt.Errorf("unmarshal %s payload: %w", record.Type, err)
	}
	if err := payload.Validate(); err != nil {
		return err
	}

	return nil
}

type recordPayload interface {
	Validate() error
}

func validateRecordOwner(userID int64) error {
	if userID <= 0 {
		return errInvalidRecordOwner
	}

	return nil
}
