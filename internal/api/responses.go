package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	type errorResponse struct {
		Status  string `json:"status"`
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}
	resp := errorResponse{
		Status: "error",
		Error:  message,
	}

	if err != nil {
		resp.Details = err.Error()
	}

	respondJSON(w, status, resp)
}
