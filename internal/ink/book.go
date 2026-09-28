package ink

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/wyw14/solo-038/internal/model"
)

type BatchBook struct {
	mu        sync.Mutex
	state     *State
	batches   map[model.BatchID]*Batch
	byCircuit map[string]model.BatchID
}

func NewBatchBook(state *State) *BatchBook {
	return &BatchBook{state: state, batches: map[model.BatchID]*Batch{}, byCircuit: map[string]model.BatchID{}}
}

func (book *BatchBook) Create(color string, recipe []PigmentRatio, plannedKg float64, now time.Time) (Batch, error) {
	batch, err := NewBatch(color, recipe, plannedKg, now)
	if err != nil {
		return Batch{}, err
	}
	book.mu.Lock()
	defer book.mu.Unlock()
	book.batches[batch.ID] = &batch
	return batch.copy(), nil
}

func (book *BatchBook) Weigh(id model.BatchID, pigment string, kg float64, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchMixing {
		return Batch{}, fmt.Errorf("ink batch %s is %s; record a refill instead of a weighing", id, batch.Status)
	}
	if err := batch.addWeighing(pigment, kg, now); err != nil {
		return Batch{}, err
	}
	batch.record("weighed", pigment, now)
	return batch.copy(), nil
}

func (book *BatchBook) Confirm(id model.BatchID, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchMixing {
		return Batch{}, fmt.Errorf("ink batch %s cannot confirm from %s", id, batch.Status)
	}
	if missing := batch.missingPigments(); len(missing) > 0 {
		return Batch{}, fmt.Errorf("ink batch %s is missing weighings for %v", id, missing)
	}
	if batch.TotalWeighed() <= 0 {
		return Batch{}, fmt.Errorf("ink batch %s has no ink weighed", id)
	}
	batch.Status = BatchConfirmed
	batch.record("confirmed", "", now)
	return batch.copy(), nil
}

func (book *BatchBook) Bind(id model.BatchID, circuit string, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchConfirmed {
		return Batch{}, fmt.Errorf("ink batch %s cannot bind from %s", id, batch.Status)
	}
	if book.state.Get(circuit) == "" {
		return Batch{}, fmt.Errorf("ink circuit %s does not exist", circuit)
	}
	if occupant, taken := book.byCircuit[circuit]; taken {
		return Batch{}, fmt.Errorf("ink circuit %s is already bound to batch %s", circuit, occupant)
	}
	batch.Status = BatchBound
	batch.Circuit = circuit
	book.byCircuit[circuit] = batch.ID
	batch.record("bound", circuit, now)
	return batch.copy(), nil
}

func (book *BatchBook) Refill(id model.BatchID, weighings []Weighing, note string, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchConfirmed && batch.Status != BatchBound {
		return Batch{}, fmt.Errorf("ink batch %s cannot refill from %s", id, batch.Status)
	}
	if len(weighings) == 0 {
		return Batch{}, fmt.Errorf("ink batch %s refill requires at least one weighing", id)
	}
	for _, weighing := range weighings {
		if err := batch.addWeighing(weighing.Pigment, weighing.Kg, now); err != nil {
			return Batch{}, err
		}
	}
	batch.record("refilled", note, now)
	return batch.copy(), nil
}

func (book *BatchBook) Scrap(id model.BatchID, destination string, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status == BatchScrapped {
		return Batch{}, fmt.Errorf("ink batch %s is already scrapped", id)
	}
	if destination == "" {
		return Batch{}, fmt.Errorf("ink batch %s scrap destination is required", id)
	}
	if batch.Circuit != "" {
		delete(book.byCircuit, batch.Circuit)
	}
	batch.Status = BatchScrapped
	batch.Destination = destination
	batch.record("scrapped", destination, now)
	return batch.copy(), nil
}

func (book *BatchBook) Get(id model.BatchID) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.lookup(id)
	if err != nil {
		return Batch{}, err
	}
	return batch.copy(), nil
}

func (book *BatchBook) List() []Batch {
	book.mu.Lock()
	defer book.mu.Unlock()
	batches := make([]Batch, 0, len(book.batches))
	for _, batch := range book.batches {
		batches = append(batches, batch.copy())
	}
	sort.Slice(batches, func(left, right int) bool {
		if batches[left].CreatedAt.Equal(batches[right].CreatedAt) {
			return batches[left].ID < batches[right].ID
		}
		return batches[left].CreatedAt.Before(batches[right].CreatedAt)
	})
	return batches
}

func (book *BatchBook) lookup(id model.BatchID) (*Batch, error) {
	if !id.Valid() {
		return nil, fmt.Errorf("ink batch id %q is invalid", id)
	}
	batch, ok := book.batches[id]
	if !ok {
		return nil, fmt.Errorf("ink batch %s does not exist", id)
	}
	return batch, nil
}
