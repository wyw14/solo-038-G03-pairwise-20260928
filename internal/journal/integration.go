package journal

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-038/internal/ink"
	"github.com/wyw14/solo-038/internal/model"
)

type IntervalRecord struct {
	Interval model.Interval `json:"interval"`
	Proofs   []model.Proof  `json:"proofs"`
	Released bool           `json:"released"`
}

func PersistInterval(store *Store, job model.Job, operation model.OperationID, record IntervalRecord, now time.Time) (uint64, error) {
	if err := record.Interval.Validate(); err != nil {
		return 0, err
	}
	event, err := model.NewEvent(operation, job, "interval_committed", record, now)
	if err != nil {
		return 0, err
	}
	sequence, err := store.Append(event)
	if err != nil {
		return sequence, err
	}
	if !store.State().IsDurable(sequence) {
		return sequence, fmt.Errorf("interval sequence %d is not durable", sequence)
	}
	return sequence, nil
}

func PersistTransition(store *Store, job model.Job, operation model.OperationID, from, to model.JobState, now time.Time) (uint64, error) {
	payload := map[string]any{"from": from, "to": to, "generation": job.Generation}
	event, err := model.NewEvent(operation, job, "job_transitioned", payload, now)
	if err != nil {
		return 0, err
	}
	return store.Append(event)
}

func PersistFence(store *Store, job model.Job, operation model.OperationID, owner string, now time.Time) (uint64, error) {
	payload := map[string]any{"owner": owner, "roll_generation": job.RollGeneration}
	event, err := model.NewEvent(operation, job, "web_fence_restored", payload, now)
	if err != nil {
		return 0, err
	}
	return store.Append(event)
}

func PersistBatch(store *Store, job model.Job, operation model.OperationID, batch ink.Batch, now time.Time) (uint64, error) {
	event, err := model.NewEvent(operation, job, "ink_batch_recorded", batch, now)
	if err != nil {
		return 0, err
	}
	sequence, err := store.Append(event)
	if err != nil {
		return sequence, err
	}
	if !store.State().IsDurable(sequence) {
		return sequence, fmt.Errorf("ink batch sequence %d is not durable", sequence)
	}
	return sequence, nil
}
