package api

import (
	"HellgameProject/internal/engine"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

type Handler struct {
	Sim *engine.WorldSimulator
}

// RegisterHandlers регистрирует все HTTP endpoints
func (h *Handler) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/api/simulate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			respondError(w, "Method Not Allowed", "Use POST for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		h.handleSimulate(w, r)
	})

	mux.HandleFunc("/api/world/state", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, "Method Not Allowed", "Use GET for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		h.handleGetWorldState(w, r)
	})

	mux.HandleFunc("/api/world/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, "Method Not Allowed", "Use GET for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		h.handleGetEvents(w, r)
	})

	mux.HandleFunc("/api/factions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, "Method Not Allowed", "Use GET for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		h.handleGetFactions(w, r)
	})

	mux.HandleFunc("/api/domains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, "Method Not Allowed", "Use GET for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		h.handleGetDomains(w, r)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondError(w, "Method Not Allowed", "Use GET for this endpoint", http.StatusMethodNotAllowed)
			return
		}
		handleHealth(w, r)
	})
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
		respondError(w, "Invalid request", err.Error(), http.StatusBadRequest)
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

	log.Printf("✅ Simulated %d ticks, generated %d events", req.Hours, len(delta.Events))
}

// handleGetWorldState - получить текущее состояние мира
func (h *Handler) handleGetWorldState(w http.ResponseWriter, r *http.Request) {
	h.Sim.Mu.RLock()
	defer h.Sim.Mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"time":           h.Sim.State.GlobalTick,
		"factions":       h.Sim.CopyFactionStates(),
		"domains":        h.Sim.CopyDomainStates(),
		"event_log_size": len(h.Sim.State.EventLog),
		"wars":           h.Sim.CopyWars(),
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

	h.Sim.Mu.RLock()
	defer h.Sim.Mu.RUnlock()

	var events []engine.GameEvent
	events = h.Sim.State.EventLog

	// Return last N events
	start := len(events) - limit
	if start < 0 {
		start = 0
	}

	if start < len(events) {
		events = events[start:]
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"count":  len(events),
		"events": events,
	})
}

// handleGetFactions - получить состояние всех фракций
func (h *Handler) handleGetFactions(w http.ResponseWriter, r *http.Request) {
	h.Sim.Mu.RLock()
	defer h.Sim.Mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"factions": h.Sim.CopyFactionStates(),
	})
}

// handleGetDomains - получить состояние всех доменов
func (h *Handler) handleGetDomains(w http.ResponseWriter, r *http.Request) {
	h.Sim.Mu.RLock()
	defer h.Sim.Mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"domains": h.Sim.CopyDomainStates(),
	})
}

// ============ RESPONSE HELPERS ============

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func respondError(w http.ResponseWriter, title, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "error",
		"error":   title,
		"message": message,
	})
	log.Printf("❌ %s: %s", title, message)
}

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
