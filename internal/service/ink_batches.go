package service

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-038/internal/ink"
	"github.com/wyw14/solo-038/internal/journal"
	"github.com/wyw14/solo-038/internal/model"
)

type InkBatchView struct {
	Batch        ink.Batch              `json:"batch"`
	Composition  map[string]float64     `json:"composition"`
	Deviation    []ink.PigmentDeviation `json:"deviation"`
	TotalKg      float64                `json:"total_weighed_kg"`
	TotalDeltaKg float64                `json:"total_delta_kg"`
	Version      uint64                 `json:"version"`
}

func inkBatchView(batch ink.Batch) InkBatchView {
	return InkBatchView{
		Batch:        batch,
		Composition:  batch.Composition(),
		Deviation:    batch.Deviation(),
		TotalKg:      batch.TotalWeighed(),
		TotalDeltaKg: batch.TotalWeighed() - batch.PlannedKg,
		Version:      batch.Version(),
	}
}

func (runtime *Runtime) CreateInkBatch(color string, recipe []ink.PigmentRatio, plannedKg float64) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Create(color, recipe, plannedKg, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) WeighInk(id model.BatchID, pigment string, kg float64) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Weigh(id, pigment, kg, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) ConfirmInkBatch(id model.BatchID) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Confirm(id, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) BindInkBatch(id model.BatchID, circuit string) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Bind(id, circuit, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) RefillInkBatch(id model.BatchID, weighings []ink.Weighing, note string) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Refill(id, weighings, note, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) ScrapInkBatch(id model.BatchID, destination string) (InkBatchView, error) {
	now := time.Now()
	batch, err := runtime.inkBatches.Scrap(id, destination, now)
	if err != nil {
		return InkBatchView{}, err
	}
	if err := runtime.persistInkBatch(batch, now); err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) InkBatch(id model.BatchID) (InkBatchView, error) {
	batch, err := runtime.inkBatches.Get(id)
	if err != nil {
		return InkBatchView{}, err
	}
	return inkBatchView(batch), nil
}

func (runtime *Runtime) InkBatches() []InkBatchView {
	batches := runtime.inkBatches.List()
	views := make([]InkBatchView, 0, len(batches))
	for _, batch := range batches {
		views = append(views, inkBatchView(batch))
	}
	return views
}

func (runtime *Runtime) persistInkBatch(batch ink.Batch, now time.Time) error {
	current, ok := runtime.jobs.Current()
	if !ok {
		return fmt.Errorf("there is no active job")
	}
	_, err := journal.PersistBatch(runtime.journal, current, model.NewOperationID(), batch, now)
	return err
}
