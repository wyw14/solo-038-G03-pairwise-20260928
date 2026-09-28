package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-038/internal/ink"
)

type batchView struct {
	ink.Batch
	Composition map[string]float64 `json:"composition"`
	Deviation   map[string]float64 `json:"deviation"`
	TotalKG     float64            `json:"total_kg"`
}

func viewBatch(batch ink.Batch) batchView {
	return batchView{Batch: batch, Composition: batch.Composition(), Deviation: batch.Deviation(), TotalKG: batch.TotalKG()}
}

type createBatchRequest struct {
	Color     string             `json:"color"`
	Targets   map[string]float64 `json:"targets"`
	PlannedKG float64            `json:"planned_kg"`
}

type weighingRequest struct {
	Component string  `json:"component"`
	WeightKG  float64 `json:"weight_kg"`
}

type confirmBatchRequest struct {
	Circuit string `json:"circuit"`
}

type topUpBatchRequest struct {
	Weighings []weighingRequest `json:"weighings"`
}

type scrapBatchRequest struct {
	Destination string `json:"destination"`
}

func (handler *Handler) inkBatches(writer http.ResponseWriter, request *http.Request) {
	batches := handler.runtime.InkBatches()
	views := make([]batchView, 0, len(batches))
	for _, batch := range batches {
		views = append(views, viewBatch(batch))
	}
	writeJSON(writer, http.StatusOK, views)
}

func (handler *Handler) createInkBatch(writer http.ResponseWriter, request *http.Request) {
	var input createBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	batch, err := handler.runtime.CreateInkBatch(input.Color, input.Targets, input.PlannedKG)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusCreated, viewBatch(batch))
}

func (handler *Handler) recordWeighing(writer http.ResponseWriter, request *http.Request) {
	var input weighingRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	batch, err := handler.runtime.RecordWeighing(chi.URLParam(request, "id"), input.Component, input.WeightKG)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, viewBatch(batch))
}

func (handler *Handler) confirmInkBatch(writer http.ResponseWriter, request *http.Request) {
	var input confirmBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	batch, err := handler.runtime.ConfirmInkBatch(chi.URLParam(request, "id"), input.Circuit)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, viewBatch(batch))
}

func (handler *Handler) topUpInkBatch(writer http.ResponseWriter, request *http.Request) {
	var input topUpBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	weighings := make([]ink.Weighing, 0, len(input.Weighings))
	for _, item := range input.Weighings {
		weighings = append(weighings, ink.Weighing{Component: item.Component, WeightKG: item.WeightKG, At: time.Now().UTC()})
	}
	batch, err := handler.runtime.TopUpInkBatch(chi.URLParam(request, "id"), weighings)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, viewBatch(batch))
}

func (handler *Handler) scrapInkBatch(writer http.ResponseWriter, request *http.Request) {
	var input scrapBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	batch, err := handler.runtime.ScrapInkBatch(chi.URLParam(request, "id"), input.Destination)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, viewBatch(batch))
}
