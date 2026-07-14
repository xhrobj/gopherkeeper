package usecase

import (
	"errors"
	"fmt"
	"sort"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var errDuplicateSyncRecordID = errors.New("duplicate record ID in synchronization state")

type RecordState struct {
	ID       string
	Revision int64
}

type RevisionChange struct {
	Metadata      model.RecordMetadata
	LocalRevision int64
}

type SyncPlan struct {
	New       []model.RecordMetadata
	Stale     []RevisionChange
	Removed   []RecordState
	Unchanged []model.RecordMetadata
}

func buildSyncPlan(serverRecords []model.RecordMetadata, localRecords []RecordState) (SyncPlan, error) {
	serverByID, err := indexServerRecords(serverRecords)
	if err != nil {
		return SyncPlan{}, err
	}
	localByID, err := indexLocalRecords(localRecords)
	if err != nil {
		return SyncPlan{}, err
	}

	plan := SyncPlan{
		New:       make([]model.RecordMetadata, 0),
		Stale:     make([]RevisionChange, 0),
		Removed:   make([]RecordState, 0),
		Unchanged: make([]model.RecordMetadata, 0),
	}

	for _, id := range sortedKeys(serverByID) {
		metadata := serverByID[id]
		local, exists := localByID[id]
		if !exists {
			plan.New = append(plan.New, metadata)
			continue
		}

		if metadata.Revision == local.Revision {
			plan.Unchanged = append(plan.Unchanged, metadata)
			continue
		}

		plan.Stale = append(plan.Stale, RevisionChange{
			Metadata:      metadata,
			LocalRevision: local.Revision,
		})
	}

	for _, id := range sortedKeys(localByID) {
		if _, exists := serverByID[id]; !exists {
			plan.Removed = append(plan.Removed, localByID[id])
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
