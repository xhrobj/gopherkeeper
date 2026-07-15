package usecase

import (
	"errors"
	"fmt"
	"sort"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var errDuplicateSyncRecordID = errors.New("duplicate record ID in synchronization state")

// RecordState содержит открытое локальное состояние записи, достаточное для сравнения ревизий.
type RecordState struct {
	// ID содержит UUID локальной записи.
	ID string

	// Revision содержит текущую локальную ревизию записи.
	Revision int64
}

// RevisionChange описывает несовпадение server и local revision одной записи.
type RevisionChange struct {
	// Metadata содержит актуальные открытые поля записи на Сервере.
	Metadata model.RecordMetadata

	// LocalRevision содержит ревизию записи в локальном кеше.
	LocalRevision int64
}

type syncPlan struct {
	newRecords []model.RecordMetadata
	stale      []RevisionChange
	removed    []RecordState
	unchanged  int
}

func buildSyncPlan(serverRecords []model.RecordMetadata, localRecords []RecordState) (syncPlan, error) {
	serverByID, err := indexServerRecords(serverRecords)
	if err != nil {
		return syncPlan{}, err
	}
	localByID, err := indexLocalRecords(localRecords)
	if err != nil {
		return syncPlan{}, err
	}

	plan := syncPlan{
		newRecords: make([]model.RecordMetadata, 0),
		stale:      make([]RevisionChange, 0),
		removed:    make([]RecordState, 0),
	}

	for _, id := range sortedKeys(serverByID) {
		metadata := serverByID[id]
		local, exists := localByID[id]
		if !exists {
			plan.newRecords = append(plan.newRecords, metadata)
			continue
		}

		if metadata.Revision == local.Revision {
			plan.unchanged++
			continue
		}

		plan.stale = append(plan.stale, RevisionChange{
			Metadata:      metadata,
			LocalRevision: local.Revision,
		})
	}

	for _, id := range sortedKeys(localByID) {
		if _, exists := serverByID[id]; !exists {
			plan.removed = append(plan.removed, localByID[id])
		}
	}

	return plan, nil
}

func indexServerRecords(records []model.RecordMetadata) (map[string]model.RecordMetadata, error) {
	indexed := make(map[string]model.RecordMetadata, len(records))
	for index, metadata := range records {
		if err := metadata.Validate(); err != nil {
			return nil, fmt.Errorf("validate server record %d: %w", index, err)
		}
		if _, exists := indexed[metadata.ID]; exists {
			return nil, fmt.Errorf("%w: server record %s", errDuplicateSyncRecordID, metadata.ID)
		}
		indexed[metadata.ID] = metadata
	}

	return indexed, nil
}

func indexLocalRecords(records []RecordState) (map[string]RecordState, error) {
	indexed := make(map[string]RecordState, len(records))
	for index, state := range records {
		if err := model.ValidateRecordID(state.ID); err != nil {
			return nil, fmt.Errorf("validate local record %d: %w", index, err)
		}
		if err := model.ValidateRecordRevision(state.Revision); err != nil {
			return nil, fmt.Errorf("validate local record %d: %w", index, err)
		}
		if _, exists := indexed[state.ID]; exists {
			return nil, fmt.Errorf("%w: local record %s", errDuplicateSyncRecordID, state.ID)
		}
		indexed[state.ID] = state
	}

	return indexed, nil
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return keys
}
