package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
)

const cacheMetadataFormatVersion = 1

type cacheMetadata struct {
	formatVersion int
	accountID     string
	kdfVersion    int
	kdfSalt       []byte
	cryptoVersion int
	keyCheckNonce []byte
	keyCheckData  []byte
}

func openCacheCrypto(
	ctx context.Context,
	database *Database,
	password []byte,
) (*cachecrypto.Service, error) {
	metadata, err := readCacheMetadata(ctx, database.db)
	if errors.Is(err, sql.ErrNoRows) {
		if !database.created {
			return nil, ErrOpenEncryptedCache
		}

		return initializeCacheMetadata(ctx, database, password)
	}
	if err != nil {
		return nil, fmt.Errorf("read local cache metadata: %w", err)
	}

	return unlockCache(metadata, database.location.AccountID, password)
}

func readCacheMetadata(ctx context.Context, database *sql.DB) (cacheMetadata, error) {
	const query = `
SELECT
    format_version,
    account_id,
    kdf_version,
    kdf_salt,
    crypto_version,
    key_check_nonce,
    key_check_ciphertext
FROM cache_metadata
WHERE singleton = 1`

	var metadata cacheMetadata
	err := database.QueryRowContext(ctx, query).Scan(
		&metadata.formatVersion,
		&metadata.accountID,
		&metadata.kdfVersion,
		&metadata.kdfSalt,
		&metadata.cryptoVersion,
		&metadata.keyCheckNonce,
		&metadata.keyCheckData,
	)
	if err != nil {
		return cacheMetadata{}, err
	}

	return metadata, nil
}

func initializeCacheMetadata(
	ctx context.Context,
	database *Database,
	password []byte,
) (*cachecrypto.Service, error) {
	salt, err := cachecrypto.GenerateSalt()
	if err != nil {
		return nil, err
	}

	key, err := cachecrypto.DeriveKey(password, salt, cachecrypto.KDFVersion)
	if err != nil {
		return nil, err
	}

	service, err := cachecrypto.NewService(key)
	if err != nil {
		return nil, err
	}

	keyCheck, err := service.CreateKeyCheck(database.location.AccountID)
	if err != nil {
		return nil, err
	}

	const insert = `
INSERT INTO cache_metadata (
    singleton,
    format_version,
    account_id,
    kdf_version,
    kdf_salt,
    crypto_version,
    key_check_nonce,
    key_check_ciphertext,
    created_at
) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?)`

	if _, err := database.db.ExecContext(
		ctx,
		insert,
		cacheMetadataFormatVersion,
		database.location.AccountID,
		cachecrypto.KDFVersion,
		salt,
		keyCheck.CryptoVersion,
		keyCheck.Nonce,
		keyCheck.Ciphertext,
		time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return nil, fmt.Errorf("create local cache metadata: %w", err)
	}

	return service, nil
}

func unlockCache(
	metadata cacheMetadata,
	accountID string,
	password []byte,
) (*cachecrypto.Service, error) {
	if metadata.formatVersion != cacheMetadataFormatVersion {
		return nil, fmt.Errorf(
			"%w: %d",
			ErrUnsupportedCacheMetadataVersion,
			metadata.formatVersion,
		)
	}
	if metadata.kdfVersion != int(cachecrypto.KDFVersion) {
		return nil, fmt.Errorf(
			"%w: %d",
			cachecrypto.ErrUnsupportedKDFVersion,
			metadata.kdfVersion,
		)
	}
	if metadata.cryptoVersion != int(cachecrypto.CryptoVersion) {
		return nil, fmt.Errorf(
			"%w: %d",
			ErrUnsupportedCacheCryptoVersion,
			metadata.cryptoVersion,
		)
	}
	if metadata.accountID != accountID {
		return nil, ErrOpenEncryptedCache
	}

	key, err := cachecrypto.DeriveKey(password, metadata.kdfSalt, cachecrypto.KDFVersion)
	if err != nil {
		return nil, ErrOpenEncryptedCache
	}

	service, err := cachecrypto.NewService(key)
	if err != nil {
		return nil, ErrOpenEncryptedCache
	}

	keyCheck := cachecrypto.EncryptedData{
		CryptoVersion: uint8(metadata.cryptoVersion),
		Nonce:         metadata.keyCheckNonce,
		Ciphertext:    metadata.keyCheckData,
	}
	if err := service.VerifyKeyCheck(accountID, keyCheck); err != nil {
		return nil, ErrOpenEncryptedCache
	}

	return service, nil
}
