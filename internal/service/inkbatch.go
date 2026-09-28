package service

import (
	"fmt"
	"time"

	"github.com/wyw14/solo-038/internal/ink"
)

func (runtime *Runtime) CreateInkBatch(color string, targets map[string]float64, plannedKG float64) (ink.Batch, error) {
	return runtime.inkBatches.Create(color, targets, plannedKG, time.Now())
}

func (runtime *Runtime) RecordWeighing(batchID, component string, weightKG float64) (ink.Batch, error) {
	return runtime.inkBatches.AddWeighing(batchID, component, weightKG, time.Now())
}

func (runtime *Runtime) ConfirmInkBatch(batchID, circuit string) (ink.Batch, error) {
	if !runtime.circuitExists(circuit) {
		return ink.Batch{}, fmt.Errorf("ink circuit %s is unknown", circuit)
	}
	return runtime.inkBatches.Confirm(batchID, circuit, time.Now())
}

func (runtime *Runtime) TopUpInkBatch(batchID string, weighings []ink.Weighing) (ink.Batch, error) {
	return runtime.inkBatches.TopUp(batchID, weighings, time.Now())
}

func (runtime *Runtime) ScrapInkBatch(batchID, destination string) (ink.Batch, error) {
	return runtime.inkBatches.Scrap(batchID, destination, time.Now())
}

func (runtime *Runtime) InkBatches() []ink.Batch {
	return runtime.inkBatches.List()
}

func (runtime *Runtime) circuitExists(circuit string) bool {
	for _, color := range runtime.ink.Colors() {
		if color == circuit {
			return true
		}
	}
	return false
}
