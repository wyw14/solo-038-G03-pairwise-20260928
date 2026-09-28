package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/wyw14/solo-038/internal/ink"
	"github.com/wyw14/solo-038/internal/model"
)

type createInkBatchRequest struct {
	Color     string             `json:"color"`
	Recipe    []ink.PigmentRatio `json:"recipe"`
	PlannedKg float64            `json:"planned_kg"`
}

type weighInkRequest struct {
	Pigment string  `json:"pigment"`
	Kg      float64 `json:"kg"`
}

type bindInkBatchRequest struct {
	Circuit string `json:"circuit"`
}

type refillInkBatchRequest struct {
	Weighings []weighInkRequest `json:"weighings"`
	Note      string            `json:"note"`
}

type scrapInkBatchRequest struct {
	Destination string `json:"destination"`
}

func (handler *Handler) inkBatches(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusOK, handler.runtime.InkBatches())
}

func (handler *Handler) createInkBatch(writer http.ResponseWriter, request *http.Request) {
	var input createInkBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	view, err := handler.runtime.CreateInkBatch(input.Color, input.Recipe, input.PlannedKg)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusCreated, view)
}

func (handler *Handler) inkBatch(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	view, err := handler.runtime.InkBatch(id)
	if err != nil {
		writeError(writer, http.StatusNotFound, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (handler *Handler) weighInk(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	var input weighInkRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	view, err := handler.runtime.WeighInk(id, input.Pigment, input.Kg)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (handler *Handler) confirmInkBatch(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	view, err := handler.runtime.ConfirmInkBatch(id)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (handler *Handler) bindInkBatch(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	var input bindInkBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	view, err := handler.runtime.BindInkBatch(id, input.Circuit)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (handler *Handler) refillInkBatch(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	var input refillInkBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	weighings := make([]ink.Weighing, 0, len(input.Weighings))
	for _, weighed := range input.Weighings {
		weighings = append(weighings, ink.Weighing{Pigment: weighed.Pigment, Kg: weighed.Kg})
	}
	view, err := handler.runtime.RefillInkBatch(id, weighings, input.Note)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func (handler *Handler) scrapInkBatch(writer http.ResponseWriter, request *http.Request) {
	id, ok := batchIDParam(writer, request)
	if !ok {
		return
	}
	var input scrapInkBatchRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, err)
		return
	}
	view, err := handler.runtime.ScrapInkBatch(id, input.Destination)
	if err != nil {
		writeError(writer, http.StatusConflict, err)
		return
	}
	writeJSON(writer, http.StatusOK, view)
}

func batchIDParam(writer http.ResponseWriter, request *http.Request) (model.BatchID, bool) {
	id := model.BatchID(chi.URLParam(request, "id"))
	if !id.Valid() {
		writeError(writer, http.StatusBadRequest, fmt.Errorf("ink batch id %q is invalid", id))
		return "", false
	}
	return id, true
}
