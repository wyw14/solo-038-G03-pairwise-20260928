package ink

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type BatchStatus string

const (
	BatchMixing    BatchStatus = "mixing"
	BatchConfirmed BatchStatus = "confirmed"
	BatchScrapped  BatchStatus = "scrapped"
)

type Weighing struct {
	Component string    `json:"component"`
	WeightKG  float64   `json:"weight_kg"`
	At        time.Time `json:"at"`
}

type BatchVersion struct {
	Version     int                `json:"version"`
	Reason      string             `json:"reason"`
	Composition map[string]float64 `json:"composition"`
	TotalKG     float64            `json:"total_kg"`
	Deviation   map[string]float64 `json:"deviation"`
	At          time.Time          `json:"at"`
}

type Batch struct {
	ID          string             `json:"id"`
	Color       string             `json:"color"`
	Targets     map[string]float64 `json:"targets"`
	PlannedKG   float64            `json:"planned_kg"`
	Weighings   []Weighing         `json:"weighings"`
	Versions    []BatchVersion     `json:"versions"`
	Status      BatchStatus        `json:"status"`
	Circuit     string             `json:"circuit,omitempty"`
	Destination string             `json:"destination,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

func (batch *Batch) TotalKG() float64 {
	total := 0.0
	for _, weighing := range batch.Weighings {
		total += weighing.WeightKG
	}
	return total
}

func (batch *Batch) Composition() map[string]float64 {
	composition := map[string]float64{}
	total := batch.TotalKG()
	if total <= 0 {
		return composition
	}
	for _, weighing := range batch.Weighings {
		composition[weighing.Component] += weighing.WeightKG / total
	}
	return composition
}

func (batch *Batch) Deviation() map[string]float64 {
	deviation := map[string]float64{}
	composition := batch.Composition()
	for component, target := range batch.Targets {
		deviation[component] = composition[component] - target
	}
	return deviation
}

func (batch *Batch) snapshot(reason string, now time.Time) BatchVersion {
	return BatchVersion{Version: len(batch.Versions) + 1, Reason: reason, Composition: batch.Composition(), TotalKG: batch.TotalKG(), Deviation: batch.Deviation(), At: now.UTC()}
}

func (batch *Batch) copy() Batch {
	clone := *batch
	clone.Targets = copyTargets(batch.Targets)
	clone.Weighings = append([]Weighing(nil), batch.Weighings...)
	clone.Versions = append([]BatchVersion(nil), batch.Versions...)
	return clone
}

func copyTargets(targets map[string]float64) map[string]float64 {
	clone := make(map[string]float64, len(targets))
	for component, fraction := range targets {
		clone[component] = fraction
	}
	return clone
}

func validateTargets(targets map[string]float64) error {
	if len(targets) == 0 {
		return fmt.Errorf("ink batch targets are required")
	}
	total := 0.0
	for component, fraction := range targets {
		if component == "" || fraction <= 0 {
			return fmt.Errorf("ink batch target for component %q is invalid", component)
		}
		total += fraction
	}
	if total < 0.999999 || total > 1.000001 {
		return fmt.Errorf("ink batch targets must sum to 1, got %f", total)
	}
	return nil
}

type Book struct {
	mu      sync.Mutex
	batches map[string]*Batch
}

func NewBook() *Book {
	return &Book{batches: map[string]*Batch{}}
}

func (book *Book) Create(color string, targets map[string]float64, plannedKG float64, now time.Time) (Batch, error) {
	if color == "" {
		return Batch{}, fmt.Errorf("ink batch color is required")
	}
	if err := validateTargets(targets); err != nil {
		return Batch{}, err
	}
	if plannedKG <= 0 {
		return Batch{}, fmt.Errorf("ink batch planned total must be positive")
	}
	book.mu.Lock()
	defer book.mu.Unlock()
	batch := &Batch{ID: uuid.NewString(), Color: color, Targets: copyTargets(targets), PlannedKG: plannedKG, Status: BatchMixing, CreatedAt: now.UTC()}
	book.batches[batch.ID] = batch
	return batch.copy(), nil
}

func (book *Book) find(id string) (*Batch, error) {
	batch, ok := book.batches[id]
	if !ok {
		return nil, fmt.Errorf("ink batch %s is unknown", id)
	}
	return batch, nil
}

func validateWeighing(batch *Batch, component string, weightKG float64) error {
	if _, ok := batch.Targets[component]; !ok {
		return fmt.Errorf("component %q is not in the ink batch recipe", component)
	}
	if weightKG <= 0 {
		return fmt.Errorf("weighing for component %q must be positive", component)
	}
	return nil
}

func (book *Book) AddWeighing(id, component string, weightKG float64, now time.Time) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.find(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchMixing {
		return Batch{}, fmt.Errorf("ink batch %s is %s and cannot take weighings", id, batch.Status)
	}
	if err := validateWeighing(batch, component, weightKG); err != nil {
		return Batch{}, err
	}
	batch.Weighings = append(batch.Weighings, Weighing{Component: component, WeightKG: weightKG, At: now.UTC()})
	return batch.copy(), nil
}

func (book *Book) Confirm(id, circuit string, now time.Time) (Batch, error) {
	if circuit == "" {
		return Batch{}, fmt.Errorf("ink batch circuit is required")
	}
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.find(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchMixing {
		return Batch{}, fmt.Errorf("ink batch %s is %s and cannot be confirmed", id, batch.Status)
	}
	if len(batch.Weighings) == 0 {
		return Batch{}, fmt.Errorf("ink batch %s has no weighings", id)
	}
	for _, other := range book.batches {
		if other.ID != id && other.Status == BatchConfirmed && other.Circuit == circuit {
			return Batch{}, fmt.Errorf("ink circuit %s already has confirmed batch %s", circuit, other.ID)
		}
	}
	batch.Status = BatchConfirmed
	batch.Circuit = circuit
	batch.Versions = append(batch.Versions, batch.snapshot("confirm", now))
	return batch.copy(), nil
}

func (book *Book) TopUp(id string, weighings []Weighing, now time.Time) (Batch, error) {
	if len(weighings) == 0 {
		return Batch{}, fmt.Errorf("ink batch top-up requires at least one weighing")
	}
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.find(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status != BatchConfirmed {
		return Batch{}, fmt.Errorf("ink batch %s is %s and cannot be topped up", id, batch.Status)
	}
	for _, weighing := range weighings {
		if err := validateWeighing(batch, weighing.Component, weighing.WeightKG); err != nil {
			return Batch{}, err
		}
	}
	for _, weighing := range weighings {
		weighing.At = now.UTC()
		batch.Weighings = append(batch.Weighings, weighing)
	}
	batch.Versions = append(batch.Versions, batch.snapshot("topup", now))
	return batch.copy(), nil
}

func (book *Book) Scrap(id, destination string, now time.Time) (Batch, error) {
	if destination == "" {
		return Batch{}, fmt.Errorf("ink batch scrap destination is required")
	}
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.find(id)
	if err != nil {
		return Batch{}, err
	}
	if batch.Status == BatchScrapped {
		return Batch{}, fmt.Errorf("ink batch %s is already scrapped", id)
	}
	batch.Status = BatchScrapped
	batch.Destination = destination
	batch.Versions = append(batch.Versions, batch.snapshot("scrap", now))
	return batch.copy(), nil
}

func (book *Book) Get(id string) (Batch, error) {
	book.mu.Lock()
	defer book.mu.Unlock()
	batch, err := book.find(id)
	if err != nil {
		return Batch{}, err
	}
	return batch.copy(), nil
}

func (book *Book) List() []Batch {
	book.mu.Lock()
	defer book.mu.Unlock()
	batches := make([]Batch, 0, len(book.batches))
	for _, batch := range book.batches {
		batches = append(batches, batch.copy())
	}
	sort.Slice(batches, func(left, right int) bool { return batches[left].CreatedAt.Before(batches[right].CreatedAt) })
	return batches
}
