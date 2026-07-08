package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const maxRequestBodySize = model.HTTPRequestBodyMaxSize

var errMultipleJSONValues = errors.New("request body must contain one JSON value")

func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	defer func() {
		_ = r.Body.Close()
	}()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errMultipleJSONValues
		}

		return err
	}

	return nil
}

func isJSONContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == "application/json"
}

func isRequestBodyTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}
