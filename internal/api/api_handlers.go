package api

import (
	"HellgameProject/internal/engine"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler struct {
	Sim engine.GameEngine //
}

func (h *Handler) NewRouter() http.Handler {
	r := chi.NewRouter()
	// Подключаем базовые middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders: []string{},
	}))

	// Прерываем чрезвычайно долгие запросы
	r.Use(middleware.Timeout(30 * time.Second))

	// Регистрируем эндпоинты
	r.Get("/health", handleHealth)
	r.Route("/api", func(r chi.Router) {
		r.Post("/simulate", h.handleSimulate)
		r.Get("/factions", h.handleGetFactions)
		r.Get("/domains", h.handleGetDomains)

		r.Route("/world", func(r chi.Router) {
			r.Get("/state", h.handleGetWorldState)
			r.Get("/events", h.handleGetEvents)
		})
	})
	// Prometheus эндпоинт для метрик
	r.Handle("/metrics", promhttp.Handler())
	return r
}

// ============ HANDLERS ============

func handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// handleSimulate - основной endpoint для запуска симуляции
func (h *Handler) handleSimulate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hours int64 `json:"hours"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if req.Hours <= 0 {
		req.Hours = 1
	}
	if req.Hours > 1000 { // Limit to prevent abuse
		req.Hours = 1000
	}

	delta := h.Sim.Simulate(req.Hours)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"ticks_simulated": delta.TicksSimulated,
		"current_time":    delta.GlobalTick,
		"events_count":    len(delta.Events),
		"events":          delta.Events,
		"factions":        delta.FactionStates,
		"domains":         delta.DomainStates,
	})

	log.Printf("Simulated %d ticks, generated %d events", req.Hours, len(delta.Events))
}

// handleGetWorldState - получить текущее состояние мира
func (h *Handler) handleGetWorldState(w http.ResponseWriter, r *http.Request) {
	state := h.Sim.GetWorldState()

	respondJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"time":           state.Time,
		"factions":       state.Factions,
		"domains":        state.Domains,
		"event_log_size": len(state.EventLog),
		"wars":           state.Wars,
	})
}

// handleGetEvents - получить события из лога
func (h *Handler) handleGetEvents(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	events := h.Sim.GetEvents(limit)

	respondJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"count":  len(events),
		"events": events,
	})
}

// handleGetFactions - получить состояние всех фракций
func (h *Handler) handleGetFactions(w http.ResponseWriter, r *http.Request) {

	factions := h.Sim.GetFactions()
	respondJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"factions": factions,
	})
}

// handleGetDomains - получить состояние всех доменов
func (h *Handler) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	domains := h.Sim.GetDomains()

	respondJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"domains": domains,
	})
}
