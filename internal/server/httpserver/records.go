package httpserver

import (
	"context"
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

var revisionETagPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

// RecordManager выполняет серверные сценарии приватных записей.
type RecordManager interface {
	// CreateText создаёт text-запись пользователя.
	CreateText(ctx context.Context, request service.CreateTextRecordRequest) (service.TextRecord, error)

	// List возвращает открытые поля записей пользователя.
	List(ctx context.Context, userID int64) ([]model.RecordMetadata, error)

	// GetText возвращает text-запись пользователя.
	GetText(ctx context.Context, userID int64, recordID string) (service.TextRecord, error)

	// UpdateText изменяет text-запись пользователя.
	UpdateText(ctx context.Context, request service.UpdateTextRecordRequest) (service.TextRecord, error)

	// Delete удаляет приватную запись пользователя.
	Delete(ctx context.Context, request service.DeleteRecordRequest) error
}

type createRecordRequest struct {
	Type    model.RecordType  `json:"type"`
	Title   string            `json:"title"`
	Payload model.TextPayload `json:"payload"`
}

type updateRecordRequest struct {
	Type    model.RecordType  `json:"type"`
	Title   string            `json:"title"`
	Payload model.TextPayload `json:"payload"`
}

type recordMetadataResponse struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type textRecordResponse struct {
	ID        string            `json:"id"`
	Type      model.RecordType  `json:"type"`
	Title     string            `json:"title"`
	Revision  int64             `json:"revision"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Payload   model.TextPayload `json:"payload"`
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

		request, err := decodeCreateRecordRequest(w, r)
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

		record, err := records.CreateText(r.Context(), service.CreateTextRecordRequest{
			UserID:  userID,
			Title:   request.Title,
			Payload: request.Payload,
		})
		if err != nil {
			writeRecordError(w, err)
			return
		}

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusCreated, newTextRecordResponse(record))
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

		recordID := r.PathValue("id")
		record, err := records.GetText(r.Context(), userID, recordID)
		if err != nil {
			writeRecordError(w, err)
			return
		}

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusOK, newTextRecordResponse(record))
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

		request, err := decodeUpdateRecordRequest(w, r)
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

		record, err := records.UpdateText(r.Context(), service.UpdateTextRecordRequest{
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

		w.Header().Set("ETag", revisionETag(record.Metadata.Revision))
		writeJSONResponse(w, http.StatusOK, newTextRecordResponse(record))
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

func decodeCreateRecordRequest(w http.ResponseWriter, r *http.Request) (createRecordRequest, error) {
	var request createRecordRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		return createRecordRequest{}, err
	}

	if request.Type != model.RecordTypeText {
		return createRecordRequest{}, model.ErrRecordTypeUnsupported
	}

	return request, nil
}

func decodeUpdateRecordRequest(w http.ResponseWriter, r *http.Request) (updateRecordRequest, error) {
	var request updateRecordRequest
	if err := decodeJSONRequest(w, r, &request); err != nil {
		return updateRecordRequest{}, err
	}

	if request.Type != model.RecordTypeText {
		return updateRecordRequest{}, model.ErrRecordTypeUnsupported
	}

	return request, nil
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

func newTextRecordResponse(record service.TextRecord) textRecordResponse {
	metadata := record.Metadata

	return textRecordResponse{
		ID:        metadata.ID,
		Type:      metadata.Type,
		Title:     metadata.Title,
		Revision:  metadata.Revision,
		CreatedAt: metadata.CreatedAt,
		UpdatedAt: metadata.UpdatedAt,
		Payload:   record.Payload,
	}
}
