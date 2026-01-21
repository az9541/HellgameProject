package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// registerHandlers регистрирует все HTTP endpoints
func registerHandlers(mux *http.ServeMux, sim *WorldSimulator) {
	mux.HandleFunc("POST /api/simulate", func(w http.ResponseWriter, r *http.Request) {
		handleSimulate(w, r, sim)
	})
	mux.HandleFunc("GET /api/world/state", func(w http.ResponseWriter, r *http.Request) {
		handleGetWorldState(w, r, sim)
	})
	mux.HandleFunc("GET /api/world/events", func(w http.ResponseWriter, r *http.Request) {
		handleGetEvents(w, r, sim)
	})
	mux.HandleFunc("GET /api/factions", func(w http.ResponseWriter, r *http.Request) {
		handleGetFactions(w, r, sim)
	})
	mux.HandleFunc("GET /api/domains", func(w http.ResponseWriter, r *http.Request) {
		handleGetDomains(w, r, sim)
	})
	mux.HandleFunc("GET /health", handleHealth)
}

// ============ HANDLERS ============

func handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// handleSimulate - основной endpoint для запуска симуляции
func handleSimulate(w http.ResponseWriter, r *http.Request, sim *WorldSimulator) {
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

	delta := sim.Simulate(req.Hours)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"hours_simulated": delta.HoursSimulated,
		"current_time":    delta.GlobalTime,
		"events_count":    len(delta.Events),
		"events":          delta.Events,
		"factions":        delta.FactionStates,
		"domains":         delta.DomainStates,
	})

	log.Printf("✅ Simulated %d hours, generated %d events", req.Hours, len(delta.Events))
}

// handleGetWorldState - получить текущее состояние мира
func handleGetWorldState(w http.ResponseWriter, r *http.Request, sim *WorldSimulator) {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "ok",
		"time":           sim.GlobalTime,
		"factions":       sim.copyFactionStates(),
		"domains":        sim.copyDomainStates(),
		"event_log_size": len(sim.EventLog),
	})
}

// handleGetEvents - получить события из лога
func handleGetEvents(w http.ResponseWriter, r *http.Request, sim *WorldSimulator) {
	location := r.URL.Query().Get("location")
	limitStr := r.URL.Query().Get("limit")
	limit := 50

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	sim.mu.RLock()
	defer sim.mu.RUnlock()

	var events []WorldEvent
	if location != "" {
		// Filter by location
		for _, event := range sim.EventLog {
			if event.Location == location {
				events = append(events, event)
			}
		}
	} else {
		// Return all
		events = sim.EventLog
	}

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
func handleGetFactions(w http.ResponseWriter, r *http.Request, sim *WorldSimulator) {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"factions": sim.copyFactionStates(),
	})
}

// handleGetDomains - получить состояние всех доменов
func handleGetDomains(w http.ResponseWriter, r *http.Request, sim *WorldSimulator) {
	sim.mu.RLock()
	defer sim.mu.RUnlock()

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"domains": sim.copyDomainStates(),
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

func corsMiddleware(next http.Handler) http.Handler {
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
