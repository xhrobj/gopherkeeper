package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

type textInputFile interface {
	io.Reader
	Stat() (os.FileInfo, error)
	Close() error
}

type openTextFileFunc func(string) (textInputFile, error)

func readRequiredTextFile(path string) (string, error) {
	return readLimitedTextFile(path, "text file", model.TextPayloadMaxSize)
}

func readOptionalTextFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	return readLimitedTextFile(path, "metadata file", model.MetadataMaxSize)
}

func readLimitedTextFile(path string, description string, maxSize int64) (string, error) {
	return readLimitedTextFileWith(path, description, maxSize, func(path string) (textInputFile, error) {
		return os.Open(path)
	})
}

func readLimitedTextFileWith(
	path string,
	description string,
	maxSize int64,
	open openTextFileFunc,
) (content string, returnErr error) {
	file, err := open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", description, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("close %s: %w", description, err))
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", description, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", description)
	}
	if info.Size() > maxSize {
		return "", fmt.Errorf("%s is too large: %w", description, model.ErrPayloadTooLarge)
	}

	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return "", fmt.Errorf("read %s: %w", description, err)
	}
	if int64(len(data)) > maxSize {
		return "", fmt.Errorf("%s is too large: %w", description, model.ErrPayloadTooLarge)
	}

	return string(data), nil
}
