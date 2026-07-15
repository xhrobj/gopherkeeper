package cache

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	projectDirectoryName = "gopherkeeper"
	cacheDirectoryName   = "cache"
	databaseFileName     = "cache.db"
	identityDomain       = "gopherkeeper-local-cache-account-v1"
)

// ErrInvalidAccountIdentity означает, что Сервер или канонический login не
// образуют допустимую идентичность локального кеша.
var ErrInvalidAccountIdentity = errors.New("invalid local cache account identity")

// Location содержит детерминированное расположение локального кеша аккаунта.
type Location struct {
	// AccountID содержит SHA-256 идентификатор пары Сервер + canonical login.
	AccountID string

	// Directory содержит отдельный каталог кеша аккаунта.
	Directory string

	// DatabaseFile содержит путь к SQLite-файлу кеша аккаунта.
	DatabaseFile string
}

// ResolveLocation вычисляет идентичность и расположение локального кеша.
//
// Пустой baseDirectory означает системный пользовательский cache-каталог:
// <user-cache-dir>/gopherkeeper/cache. canonicalLogin должен уже находиться в
// каноническом lowercase-виде, возвращаемом Сервером.
func ResolveLocation(baseDirectory, serverAddress, canonicalLogin string) (Location, error) {
	return resolveLocation(baseDirectory, serverAddress, canonicalLogin, os.UserCacheDir)
}

type userCacheDirFunc func() (string, error)

func resolveLocation(
	baseDirectory string,
	serverAddress string,
	canonicalLogin string,
	userCacheDir userCacheDirFunc,
) (Location, error) {
	if userCacheDir == nil {
		return Location{}, errors.New("user cache directory resolver is required")
	}

	canonicalAddress, err := normalizeServerAddress(serverAddress)
	if err != nil {
		return Location{}, err
	}
	if err := model.ValidateCanonicalLogin(canonicalLogin); err != nil {
		return Location{}, fmt.Errorf("%w: canonical login: %w", ErrInvalidAccountIdentity, err)
	}

	resolvedBaseDirectory, err := resolveBaseDirectory(baseDirectory, userCacheDir)
	if err != nil {
		return Location{}, err
	}

	accountID := buildAccountID(canonicalAddress, canonicalLogin)
	accountDirectory := filepath.Join(resolvedBaseDirectory, accountID)

	return Location{
		AccountID:    accountID,
		Directory:    accountDirectory,
		DatabaseFile: filepath.Join(accountDirectory, databaseFileName),
	}, nil
}

func resolveBaseDirectory(baseDirectory string, userCacheDir userCacheDirFunc) (string, error) {
	if baseDirectory != "" {
		return filepath.Clean(baseDirectory), nil
	}

	cacheDirectory, err := userCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}
	if cacheDirectory == "" {
		return "", errors.New("resolve user cache directory: empty path")
	}

	return filepath.Join(cacheDirectory, projectDirectoryName, cacheDirectoryName), nil
}

func normalizeServerAddress(serverAddress string) (string, error) {
	address := strings.TrimSpace(serverAddress)
	if address == "" {
		return "", fmt.Errorf("%w: server address is required", ErrInvalidAccountIdentity)
	}

	host, portText, err := net.SplitHostPort(address)
	if err != nil || host == "" || portText == "" {
		return "", fmt.Errorf("%w: server address must use host:port", ErrInvalidAccountIdentity)
	}

	port, err := strconv.ParseUint(portText, 10, 16)
	if err != nil || port == 0 {
		return "", fmt.Errorf("%w: server port is invalid", ErrInvalidAccountIdentity)
	}

	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	} else {
		host = strings.ToLower(host)
	}

	return net.JoinHostPort(host, strconv.FormatUint(port, 10)), nil
}

func buildAccountID(serverAddress, canonicalLogin string) string {
	identity := make([]byte, 0, len(identityDomain)+len(serverAddress)+len(canonicalLogin)+3*8)
	identity = appendIdentityPart(identity, identityDomain)
	identity = appendIdentityPart(identity, serverAddress)
	identity = appendIdentityPart(identity, canonicalLogin)

	digest := sha256.Sum256(identity)
	return hex.EncodeToString(digest[:])
}

func appendIdentityPart(identity []byte, value string) []byte {
	identity = binary.BigEndian.AppendUint64(identity, uint64(len(value)))
	return append(identity, value...)
}
