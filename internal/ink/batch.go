package ink

import (
	"fmt"
	"sort"
	"time"

	"github.com/wyw14/solo-038/internal/model"
)

type BatchStatus string

const (
	BatchMixing    BatchStatus = "mixing"
	BatchConfirmed BatchStatus = "confirmed"
	BatchBound     BatchStatus = "bound"
	BatchScrapped  BatchStatus = "scrapped"
)

type PigmentRatio struct {
	Pigment  string  `json:"pigment"`
	Fraction float64 `json:"fraction"`
}

type Weighing struct {
	Pigment string    `json:"pigment"`
	Kg      float64   `json:"kg"`
	At      time.Time `json:"at"`
}

type BatchEvent struct {
	Version uint64    `json:"version"`
	Kind    string    `json:"kind"`
	Detail  string    `json:"detail,omitempty"`
	TotalKg float64   `json:"total_kg"`
	At      time.Time `json:"at"`
}

type PigmentDeviation struct {
	Pigment string  `json:"pigment"`
	Target  float64 `json:"target"`
	Actual  float64 `json:"actual"`
	Delta   float64 `json:"delta"`
}

type Batch struct {
	ID          model.BatchID  `json:"id"`
	Color       string         `json:"color"`
	Recipe      []PigmentRatio `json:"recipe"`
	PlannedKg   float64        `json:"planned_kg"`
	Weighings   []Weighing     `json:"weighings"`
	Status      BatchStatus    `json:"status"`
	Circuit     string         `json:"circuit,omitempty"`
	Destination string         `json:"destination,omitempty"`
	History     []BatchEvent   `json:"history"`
	CreatedAt   time.Time      `json:"created_at"`
}

func NewBatch(color string, recipe []PigmentRatio, plannedKg float64, now time.Time) (Batch, error) {
	if color == "" {
		return Batch{}, fmt.Errorf("ink batch color is required")
	}
	if plannedKg <= 0 {
		return Batch{}, fmt.Errorf("ink batch planned total must be positive")
	}
	if err := validateRecipe(recipe); err != nil {
		return Batch{}, err
	}
	batch := Batch{
		ID:        model.NewBatchID(),
		Color:     color,
		Recipe:    append([]PigmentRatio(nil), recipe...),
		PlannedKg: plannedKg,
		Status:    BatchMixing,
		CreatedAt: now.UTC(),
	}
	batch.record("created", "", now)
	return batch, nil
}

func validateRecipe(recipe []PigmentRatio) error {
	if len(recipe) == 0 {
		return fmt.Errorf("ink batch recipe is required")
	}
	seen := map[string]bool{}
	total := 0.0
	for _, ratio := range recipe {
		if ratio.Pigment == "" {
			return fmt.Errorf("ink batch recipe pigment is required")
		}
		if ratio.Fraction <= 0 {
			return fmt.Errorf("ink batch recipe fraction for %s must be positive", ratio.Pigment)
		}
		if seen[ratio.Pigment] {
			return fmt.Errorf("ink batch recipe repeats pigment %s", ratio.Pigment)
		}
		seen[ratio.Pigment] = true
		total += ratio.Fraction
	}
	if total < 1-1e-6 || total > 1+1e-6 {
		return fmt.Errorf("ink batch recipe fractions must sum to 1, got %f", total)
	}
	return nil
}

func (batch *Batch) record(kind, detail string, now time.Time) {
	batch.History = append(batch.History, BatchEvent{
		Version: uint64(len(batch.History)) + 1,
		Kind:    kind,
		Detail:  detail,
		TotalKg: batch.TotalWeighed(),
		At:      now.UTC(),
	})
}

func (batch *Batch) Version() uint64 {
	return uint64(len(batch.History))
}

func (batch *Batch) TotalWeighed() float64 {
	total := 0.0
	for _, weighing := range batch.Weighings {
		total += weighing.Kg
	}
	return total
}

func (batch *Batch) Composition() map[string]float64 {
	totals := map[string]float64{}
	for _, weighing := range batch.Weighings {
		totals[weighing.Pigment] += weighing.Kg
	}
	total := batch.TotalWeighed()
	composition := map[string]float64{}
	if total <= 0 {
		return composition
	}
	for pigment, kg := range totals {
		composition[pigment] = kg / total
	}
	return composition
}

func (batch *Batch) Deviation() []PigmentDeviation {
	targets := map[string]float64{}
	for _, ratio := range batch.Recipe {
		targets[ratio.Pigment] = ratio.Fraction
	}
	actual := batch.Composition()
	pigments := map[string]bool{}
	for pigment := range targets {
		pigments[pigment] = true
	}
	for pigment := range actual {
		pigments[pigment] = true
	}
	names := make([]string, 0, len(pigments))
	for pigment := range pigments {
		names = append(names, pigment)
	}
	sort.Strings(names)
	deviations := make([]PigmentDeviation, 0, len(names))
	for _, pigment := range names {
		deviations = append(deviations, PigmentDeviation{
			Pigment: pigment,
			Target:  targets[pigment],
			Actual:  actual[pigment],
			Delta:   actual[pigment] - targets[pigment],
		})
	}
	return deviations
}

func (batch *Batch) recipeHas(pigment string) bool {
	for _, ratio := range batch.Recipe {
		if ratio.Pigment == pigment {
			return true
		}
	}
	return false
}

func (batch *Batch) addWeighing(pigment string, kg float64, now time.Time) error {
	if !batch.recipeHas(pigment) {
		return fmt.Errorf("pigment %s is not in the recipe of ink batch %s", pigment, batch.ID)
	}
	if kg <= 0 {
		return fmt.Errorf("weighed amount for %s must be positive", pigment)
	}
	batch.Weighings = append(batch.Weighings, Weighing{Pigment: pigment, Kg: kg, At: now.UTC()})
	return nil
}

func (batch *Batch) missingPigments() []string {
	weighed := map[string]bool{}
	for _, weighing := range batch.Weighings {
		weighed[weighing.Pigment] = true
	}
	missing := []string{}
	for _, ratio := range batch.Recipe {
		if !weighed[ratio.Pigment] {
			missing = append(missing, ratio.Pigment)
		}
	}
	sort.Strings(missing)
	return missing
}

func (batch *Batch) copy() Batch {
	clone := *batch
	clone.Recipe = append([]PigmentRatio(nil), batch.Recipe...)
	clone.Weighings = append([]Weighing(nil), batch.Weighings...)
	clone.History = append([]BatchEvent(nil), batch.History...)
	return clone
}
