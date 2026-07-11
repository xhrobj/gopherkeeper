package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func readRecordPayloadJSON(
	input io.Reader,
	description string,
	multipleValuesError error,
	payload any,
) error {
	data, err := io.ReadAll(io.LimitReader(input, model.HTTPRequestBodyMaxSize+1))
	if err != nil {
		return fmt.Errorf("read %s from stdin: %w", description, err)
	}
	if int64(len(data)) > model.HTTPRequestBodyMaxSize {
		return fmt.Errorf("read %s from stdin: %w", description, model.ErrPayloadTooLarge)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		return fmt.Errorf("decode %s from stdin: %w", description, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return multipleValuesError
		}

		return fmt.Errorf("decode %s from stdin: %w", description, err)
	}

	return nil
}
