package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	errorCodeRecordNotFound       = "record_not_found"
	errorCodeRevisionConflict     = "record_revision_conflict"
	errorCodePreconditionRequired = "precondition_required"

	errorMessageInvalidRecordRequest = "invalid record request"
	errorMessageRecordNotFound       = "record not found"
	errorMessageRevisionConflict     = "record revision conflict"
	errorMessagePreconditionRequired = "record revision is required"
)

var (
	revisionETagPattern      = regexp.MustCompile(`^[1-9][0-9]*$`)
	errInvalidRecordResponse = errors.New("invalid record response")
)

// RecordManager выполняет серверные сценарии приватных записей.
type RecordManager interface {
	// Create создаёт приватную запись пользователя.
	Create(ctx context.Context, request service.CreateRecordRequest) (service.DecryptedRecord, error)

	// List возвращает открытые поля записей пользователя.
	List(ctx context.Context, userID int64) ([]model.RecordMetadata, error)

	// Get возвращает запись пользователя с расшифрованным payload согласно её типу.
	Get(ctx context.Context, userID int64, recordID string) (service.DecryptedRecord, error)

	// Update изменяет приватную запись пользователя.
	Update(ctx context.Context, request service.UpdateRecordRequest) (service.DecryptedRecord, error)

	// Delete удаляет приватную запись пользователя.
	Delete(ctx context.Context, request service.DeleteRecordRequest) error
}

type recordRequestEnvelope struct {
	Type    model.RecordType `json:"type"`
	Title   string           `json:"title"`
	Payload json.RawMessage  `json:"payload"`
}

type decodedRecordRequest struct {
	Title   string
	Payload model.RecordPayload
}

type recordMetadataResponse struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type recordResponse struct {
	recordMetadataResponse
	Payload model.RecordPayload `json:"payload"`
}

type listRecordsResponse struct {
	Records []recordMetadataResponse `json:"records"`
}

func createRecordHandler(records RecordManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeErrorResponse(
				w,
				http.StatusUnsupportedMediaType,
				errorCodeUnsupportedMediaType,
				errorMessageUnsupportedMediaType,
			)
			return
		}

		request, err := decodeRecordRequest(w, r)
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeErrorResponse(
					w,
					http.StatusRequestEntityTooLarge,
					errorCodePayloadTooLarge,
					errorMessagePayloadTooLarge,
				)
				return
			}

			writeInvalidRecordRequest(w)
			return
		}

		record, err := records.Create(r.Context(), service.CreateRecordRequest{
			UserID:  userID,
			Title:   request.Title,
			Payload: request.Payload,
		})
		if err != nil {
			writeRecordError(w, err)
			return
		}

		response, err := newRecordResponse(record)
		if err != nil {
			writeRecordError(w, err)
			return
		}

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusCreated, response)
	})
}

func listRecordsHandler(records RecordManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		items, err := records.List(r.Context(), userID)
		if err != nil {
			writeRecordError(w, err)
			return
		}

		response := listRecordsResponse{
			Records: make([]recordMetadataResponse, 0, len(items)),
		}
		for _, item := range items {
			response.Records = append(response.Records, newRecordMetadataResponse(item))
		}

		writeJSONResponse(w, http.StatusOK, response)
	})
}

func getRecordHandler(records RecordManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		record, err := records.Get(r.Context(), userID, r.PathValue("id"))
		if err != nil {
			writeRecordError(w, err)
			return
		}

		response, err := newRecordResponse(record)
		if err != nil {
			writeRecordError(w, err)
			return
		}

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusOK, response)
	})
}

func updateRecordHandler(records RecordManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		expectedRevision, err := parseIfMatchRevision(r.Header.Get("If-Match"))
		if err != nil {
			writeRecordError(w, err)
			return
		}

		if !isJSONContentType(r.Header.Get("Content-Type")) {
			writeErrorResponse(
				w,
				http.StatusUnsupportedMediaType,
				errorCodeUnsupportedMediaType,
				errorMessageUnsupportedMediaType,
			)
			return
		}

		request, err := decodeRecordRequest(w, r)
		if err != nil {
			if isRequestBodyTooLarge(err) {
				writeErrorResponse(
					w,
					http.StatusRequestEntityTooLarge,
					errorCodePayloadTooLarge,
					errorMessagePayloadTooLarge,
				)
				return
			}

			writeInvalidRecordRequest(w)
			return
		}

		record, err := records.Update(r.Context(), service.UpdateRecordRequest{
			UserID:           userID,
			RecordID:         r.PathValue("id"),
			ExpectedRevision: expectedRevision,
			Title:            request.Title,
			Payload:          request.Payload,
		})
		if err != nil {
			writeRecordError(w, err)
			return
		}

		response, err := newRecordResponse(record)
		if err != nil {
			writeRecordError(w, err)
			return
		}

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusOK, response)
	})
}

func deleteRecordHandler(records RecordManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middleware.UserIDFromContext(r.Context())
		if !ok {
			middleware.WriteUnauthorizedResponse(w)
			return
		}

		expectedRevision, err := parseIfMatchRevision(r.Header.Get("If-Match"))
		if err != nil {
			writeRecordError(w, err)
			return
		}

		if err := records.Delete(r.Context(), service.DeleteRecordRequest{
			UserID:           userID,
			RecordID:         r.PathValue("id"),
			ExpectedRevision: expectedRevision,
		}); err != nil {
			writeRecordError(w, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})
}

func decodeRecordRequest(w http.ResponseWriter, r *http.Request) (decodedRecordRequest, error) {
	var envelope recordRequestEnvelope
	if err := decodeJSONRequest(w, r, &envelope); err != nil {
		return decodedRecordRequest{}, err
	}

	payload, err := model.NewRecordPayload(envelope.Type)
	if err != nil {
		return decodedRecordRequest{}, err
	}
	if err := decodeJSONPayload(envelope.Payload, payload); err != nil {
		return decodedRecordRequest{}, err
	}

	return decodedRecordRequest{
		Title:   envelope.Title,
		Payload: payload,
	}, nil
}

func parseIfMatchRevision(value string) (int64, error) {
	if value == "" {
		return 0, model.ErrRecordPreconditionRequired
	}

	unquoted, err := strconv.Unquote(value)
	if err != nil || !revisionETagPattern.MatchString(unquoted) {
		return 0, model.ErrInvalidRecordRevision
	}

	revision, err := strconv.ParseInt(unquoted, 10, 64)
	if err != nil {
		return 0, model.ErrInvalidRecordRevision
	}

	return revision, nil
}

func writeRecordError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrPayloadTooLarge):
		writeErrorResponse(
			w,
			http.StatusRequestEntityTooLarge,
			errorCodePayloadTooLarge,
			errorMessagePayloadTooLarge,
		)

	case errors.Is(err, model.ErrRecordNotFound):
		writeErrorResponse(
			w,
			http.StatusNotFound,
			errorCodeRecordNotFound,
			errorMessageRecordNotFound,
		)

	case errors.Is(err, model.ErrRecordRevisionConflict):
		writeErrorResponse(
			w,
			http.StatusConflict,
			errorCodeRevisionConflict,
			errorMessageRevisionConflict,
		)

	case errors.Is(err, model.ErrRecordPreconditionRequired):
		writeErrorResponse(
			w,
			http.StatusPreconditionRequired,
			errorCodePreconditionRequired,
			errorMessagePreconditionRequired,
		)

	case errors.Is(err, model.ErrInvalidRecordID),
		errors.Is(err, model.ErrInvalidRecordRevision),
		errors.Is(err, model.ErrInvalidRecordTitle),
		errors.Is(err, model.ErrInvalidTextPayload),
		errors.Is(err, model.ErrInvalidCredentialsPayload),
		errors.Is(err, model.ErrRecordTypeUnsupported):
		writeInvalidRecordRequest(w)

	default:
		writeErrorResponse(
			w,
			http.StatusInternalServerError,
			errorCodeInternal,
			errorMessageInternal,
		)
	}
}

func writeInvalidRecordRequest(w http.ResponseWriter) {
	writeErrorResponse(
		w,
		http.StatusBadRequest,
		errorCodeInvalidRequest,
		errorMessageInvalidRecordRequest,
	)
}

func revisionETag(revision int64) string {
	return strconv.Quote(strconv.FormatInt(revision, 10))
}

func newRecordMetadataResponse(metadata model.RecordMetadata) recordMetadataResponse {
	return recordMetadataResponse{
		ID:        metadata.ID,
		Type:      metadata.Type,
		Title:     metadata.Title,
		Revision:  metadata.Revision,
		CreatedAt: metadata.CreatedAt,
		UpdatedAt: metadata.UpdatedAt,
	}
}

func newRecordResponse(record service.DecryptedRecord) (recordResponse, error) {
	if record.Payload == nil {
		return recordResponse{}, errInvalidRecordResponse
	}
	if err := record.Payload.Validate(); err != nil || record.Metadata.Type != record.Payload.RecordType() {
		return recordResponse{}, errInvalidRecordResponse
	}

	return recordResponse{
		recordMetadataResponse: newRecordMetadataResponse(record.Metadata),
		Payload:                record.Payload,
	}, nil
}
