package cli

import (
	"fmt"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

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
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", description, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", description)
	}
	if info.Size() > maxSize {
		return "", fmt.Errorf("%s is too large: %w", description, model.ErrPayloadTooLarge)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", description, err)
	}

	return string(data), nil
}
