package binaryfile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Read читает бинарный файл целиком в пределах доменного лимита и возвращает
// его базовое имя вместе с содержимым. Каталоги и файлы больше 2 МиБ
// отклоняются до передачи данных в application-слой.
func Read(path string) (string, []byte, error) {
	if path == "" {
		return "", nil, errors.New("binary file path is required")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open binary file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return "", nil, fmt.Errorf("stat binary file: %w", err)
	}
	if info.IsDir() {
		_ = file.Close()
		return "", nil, errors.New("binary file is a directory")
	}
	if info.Size() > int64(model.BinaryPayloadMaxSize) {
		_ = file.Close()
		return "", nil, fmt.Errorf("binary file is too large: %w", model.ErrPayloadTooLarge)
	}

	data, readErr := io.ReadAll(io.LimitReader(file, int64(model.BinaryPayloadMaxSize)+1))
	closeErr := file.Close()
	if readErr != nil {
		return "", nil, fmt.Errorf("read binary file: %w", readErr)
	}
	if closeErr != nil {
		return "", nil, fmt.Errorf("close binary file: %w", closeErr)
	}
	if len(data) > model.BinaryPayloadMaxSize {
		return "", nil, fmt.Errorf("binary file is too large: %w", model.ErrPayloadTooLarge)
	}
	if data == nil {
		data = []byte{}
	}

	return filepath.Base(path), data, nil
}
