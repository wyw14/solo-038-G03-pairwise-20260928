package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/wyw14/solo-038/internal/service"
)

func NewRouter(runtime *service.Runtime) http.Handler {
	handler := &Handler{runtime: runtime}
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(5 * time.Second))
	router.Get("/healthz", handler.health)
	router.Route("/api", func(api chi.Router) {
		api.Get("/operations", handler.operations)
		api.Post("/operations", handler.createOperation)
		api.Get("/equipment", handler.equipment)
		api.Get("/interlocks", handler.interlocks)
		api.Get("/incidents", handler.incidents)
		api.Get("/recovery", handler.recoveryStatus)
		api.Post("/jobs/current/cancel", handler.cancelCurrent)
		api.Route("/ink/batches", func(batches chi.Router) {
			batches.Get("/", handler.inkBatches)
			batches.Post("/", handler.createInkBatch)
			batches.Get("/{id}", handler.inkBatch)
			batches.Post("/{id}/weighings", handler.weighInk)
			batches.Post("/{id}/confirm", handler.confirmInkBatch)
			batches.Post("/{id}/bind", handler.bindInkBatch)
			batches.Post("/{id}/refill", handler.refillInkBatch)
			batches.Post("/{id}/scrap", handler.scrapInkBatch)
		})
	})
	return router
}

type Handler struct {
	runtime *service.Runtime
}

func (handler *Handler) health(writer http.ResponseWriter, request *http.Request) {
	if !handler.runtime.Healthy() {
		writeJSON(writer, http.StatusServiceUnavailable, map[string]any{"status": "unhealthy"})
		return
	}
	writeJSON(writer, http.StatusOK, map[string]any{"status": "ok", "request_id": middleware.GetReqID(request.Context())})
}
