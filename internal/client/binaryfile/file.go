package binaryfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

type outputFile interface {
	io.Writer
	io.Closer
}

// Write создаёт новый файл с правами 0600 и записывает в него data.
// Существующий файл не перезаписывается. При ошибке записи или закрытия
// незавершённый новый файл удаляется.
func Write(path string, data []byte) error {
	return writeWith(
		path,
		data,
		func(outputPath string) (outputFile, error) {
			return os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		},
		os.Remove,
	)
}

func writeWith(
	path string,
	data []byte,
	create func(string) (outputFile, error),
	remove func(string) error,
) error {
	if path == "" {
		return errors.New("output path is required")
	}

	file, err := create(path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	_, writeErr := io.Copy(file, bytes.NewReader(data))
	closeErr := file.Close()
	if writeErr == nil && closeErr == nil {
		return nil
	}

	removeErr := remove(path)
	return errors.Join(
		wrapOptionalError("write output file", writeErr),
		wrapOptionalError("close output file", closeErr),
		wrapOptionalError("remove incomplete output file", removeErr),
	)
}

func wrapOptionalError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}
