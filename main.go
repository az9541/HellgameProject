package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// Initialize simulator (the heart of the backend)
	simulator := NewWorldSimulator()
	StartEventLogger(simulator.EventBus, 200)

	// Start background goroutines for world simulation
	simulator.Start()

	// Create router
	mux := http.NewServeMux()
	handler := corsMiddleware(mux)

	// Register API handlers
	registerHandlers(mux, simulator)

	// Start server
	port := ":8080"
	fmt.Printf("🎮 Hell Game Backend (Simulator) starting on http://localhost%s\n", port)
	printEndpoints()

	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("❌ Server error: %v", err)
	}
}

func printEndpoints() {
	fmt.Println("📍 Main Endpoints:")
	fmt.Println("  POST /api/simulate          - Run simulation for N hours")
	fmt.Println("  GET  /api/world/state       - Get current world state")
	fmt.Println("  GET  /api/world/events      - Get events for domain")
	fmt.Println("  GET  /api/factions          - Get all factions")
	fmt.Println("  GET  /api/domains           - Get all domains")
	fmt.Println("  GET  /health                - Health check")
	fmt.Println("")
	fmt.Println("💡 Example:")
	fmt.Println(`  curl -X POST http://localhost:8080/api/simulate \`)
	fmt.Println(`    -H "Content-Type: application/json" \`)
	fmt.Println(`    -d '{"hours": 24}'`)
}
