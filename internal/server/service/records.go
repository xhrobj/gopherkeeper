package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

var (
	errInvalidRecordOwner  = errors.New("invalid record owner")
	errInvalidStoredRecord = errors.New("invalid stored record")
)

// RecordRepository содержит операции хранения, необходимые RecordService.
type RecordRepository interface {
	// Create сохраняет encrypted record и возвращает зафиксированное состояние.
	Create(ctx context.Context, record model.EncryptedRecord) (model.EncryptedRecord, error)

	// ListMetadata возвращает открытые поля записей пользователя без payload.
	ListMetadata(ctx context.Context, userID int64) ([]model.RecordMetadata, error)

	// Get возвращает encrypted record пользователя по идентификатору.
	Get(ctx context.Context, userID int64, recordID string) (model.EncryptedRecord, error)

	// Update изменяет encrypted record при совпадении ожидаемой ревизии.
	Update(ctx context.Context, record model.EncryptedRecord, expectedRevision int64) (model.EncryptedRecord, error)

	// Delete физически удаляет encrypted record при совпадении ожидаемой ревизии.
	Delete(ctx context.Context, userID int64, recordID string, expectedRevision int64) error
}

// RecordPayloadCrypto шифрует и расшифровывает приватные payload'ы записей.
type RecordPayloadCrypto interface {
	// Encrypt шифрует plaintext payload с authenticated data.
	Encrypt(plaintext []byte, aad []byte) (recordcrypto.EncryptedPayload, error)

	// Decrypt расшифровывает encrypted payload с authenticated data.
	Decrypt(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error)
}

// CreateRecordRequest содержит входные данные для создания приватной записи.
type CreateRecordRequest struct {
	// UserID содержит идентификатор владельца создаваемой записи.
	UserID int64

	// Title содержит открытое название создаваемой записи.
	Title string

	// Payload содержит типизированные приватные данные создаваемой записи.
	Payload model.RecordPayload
}

// UpdateRecordRequest содержит входные данные для изменения приватной записи.
type UpdateRecordRequest struct {
	// UserID содержит идентификатор владельца изменяемой записи.
	UserID int64

	// RecordID содержит UUID изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, ожидаемую Клиентом.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новые типизированные приватные данные записи.
	Payload model.RecordPayload
}

// DeleteRecordRequest содержит входные данные для удаления приватной записи.
type DeleteRecordRequest struct {
	// UserID содержит идентификатор владельца удаляемой записи.
	UserID int64

	// RecordID содержит UUID удаляемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, ожидаемую Клиентом.
	ExpectedRevision int64
}

// DecryptedRecord содержит открытую metadata и типизированный расшифрованный payload.
type DecryptedRecord struct {
	// Metadata содержит открытые поля приватной записи.
	Metadata model.RecordMetadata

	// Payload содержит типизированный расшифрованный приватный payload.
	Payload model.RecordPayload
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

// Create создаёт приватную запись, шифрует payload и сохраняет encrypted record.
func (s *RecordService) Create(ctx context.Context, request CreateRecordRequest) (DecryptedRecord, error) {
	if err := validateRecordOwner(request.UserID); err != nil {
		return DecryptedRecord{}, err
	}

	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return DecryptedRecord{}, err
	}

	if request.Payload == nil {
		return DecryptedRecord{}, model.ErrRecordTypeUnsupported
	}

	if err := request.Payload.Validate(); err != nil {
		return DecryptedRecord{}, err
	}

	recordID, err := model.NewRecordID()
	if err != nil {
		return DecryptedRecord{}, err
	}

	recordType := request.Payload.RecordType()
	encrypted, err := s.encryptRecordPayload(request.UserID, recordID, request.Payload)
	if err != nil {
		return DecryptedRecord{}, err
	}

	record, err := s.records.Create(ctx, model.EncryptedRecord{
		ID:            recordID,
		UserID:        request.UserID,
		Type:          recordType,
		Title:         request.Title,
		CryptoVersion: encrypted.CryptoVersion,
		KeyID:         encrypted.KeyID,
		Nonce:         encrypted.Nonce,
		Ciphertext:    encrypted.Ciphertext,
	})
	if err != nil {
		return DecryptedRecord{}, fmt.Errorf("create %s record: %w", recordType, err)
	}

	if record.Type != recordType {
		return DecryptedRecord{}, fmt.Errorf("%w: created record type %q", errInvalidStoredRecord, record.Type)
	}

	return DecryptedRecord{
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

// Get возвращает запись пользователя и расшифровывает payload согласно её типу.
func (s *RecordService) Get(ctx context.Context, userID int64, recordID string) (DecryptedRecord, error) {
	if err := validateRecordOwner(userID); err != nil {
		return DecryptedRecord{}, err
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return DecryptedRecord{}, err
	}

	record, err := s.records.Get(ctx, userID, recordID)
	if err != nil {
		return DecryptedRecord{}, fmt.Errorf("get record: %w", err)
	}

	payload, err := model.NewRecordPayload(record.Type)
	if err != nil {
		return DecryptedRecord{}, fmt.Errorf("%w: unsupported type %q", errInvalidStoredRecord, record.Type)
	}
	if err := s.decryptRecordPayload(record, payload); err != nil {
		return DecryptedRecord{}, err
	}

	return DecryptedRecord{
		Metadata: record.Metadata(),
		Payload:  payload,
	}, nil
}

// Update изменяет приватную запись при совпадении ожидаемой ревизии.
func (s *RecordService) Update(ctx context.Context, request UpdateRecordRequest) (DecryptedRecord, error) {
	if err := validateRecordOwner(request.UserID); err != nil {
		return DecryptedRecord{}, err
	}
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return DecryptedRecord{}, err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return DecryptedRecord{}, err
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return DecryptedRecord{}, err
	}
	if request.Payload == nil {
		return DecryptedRecord{}, model.ErrRecordTypeUnsupported
	}
	if err := request.Payload.Validate(); err != nil {
		return DecryptedRecord{}, err
	}

	recordType := request.Payload.RecordType()
	current, err := s.records.Get(ctx, request.UserID, request.RecordID)
	if err != nil {
		return DecryptedRecord{}, fmt.Errorf("get %s record for update: %w", recordType, err)
	}
	if current.Type != recordType {
		return DecryptedRecord{}, model.ErrRecordTypeUnsupported
	}
	if current.Revision != request.ExpectedRevision {
		return DecryptedRecord{}, model.ErrRecordRevisionConflict
	}

	encrypted, err := s.encryptRecordPayload(request.UserID, request.RecordID, request.Payload)
	if err != nil {
		return DecryptedRecord{}, err
	}

	updated, err := s.records.Update(ctx, model.EncryptedRecord{
		ID:            request.RecordID,
		UserID:        request.UserID,
		Type:          recordType,
		Title:         request.Title,
		CryptoVersion: encrypted.CryptoVersion,
		KeyID:         encrypted.KeyID,
		Nonce:         encrypted.Nonce,
		Ciphertext:    encrypted.Ciphertext,
	}, request.ExpectedRevision)
	if err != nil {
		return DecryptedRecord{}, fmt.Errorf("update %s record: %w", recordType, err)
	}
	if updated.Type != recordType {
		return DecryptedRecord{}, fmt.Errorf("%w: updated record type %q", errInvalidStoredRecord, updated.Type)
	}

	return DecryptedRecord{
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

func (s *RecordService) encryptRecordPayload(
	userID int64,
	recordID string,
	payload model.RecordPayload,
) (recordcrypto.EncryptedPayload, error) {
	recordType := payload.RecordType()
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

func (s *RecordService) decryptRecordPayload(record model.EncryptedRecord, payload model.RecordPayload) error {
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

func validateRecordOwner(userID int64) error {
	if userID <= 0 {
		return errInvalidRecordOwner
	}

	return nil
}
