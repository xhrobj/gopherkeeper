package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

type binaryOutputFile interface {
	io.Writer
	io.Closer
}

func readBinaryFile(path string) (string, []byte, error) {
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

func writeBinaryFile(path string, data []byte) error {
	return writeBinaryFileWith(
		path,
		data,
		func(outputPath string) (binaryOutputFile, error) {
			return os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		},
		os.Remove,
	)
}

func writeBinaryFileWith(
	path string,
	data []byte,
	create func(string) (binaryOutputFile, error),
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
