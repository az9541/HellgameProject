package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	deterministic := flag.Bool("deterministic", false, "run simulator in deterministic mode")
	seed := flag.Int64("seed", 1, "seed for deterministic mode")
	batchSeeds := flag.String("batch-seeds", "", "seed range for batch mode (N or A:B)")
	batchTicks := flag.Int64("batch-ticks", 200, "ticks per run in batch mode")
	batchOut := flag.String("batch-out", "", "output CSV path for batch mode")
	flag.Parse()

	if strings.TrimSpace(*batchSeeds) != "" || strings.TrimSpace(*batchOut) != "" {
		if strings.TrimSpace(*batchSeeds) == "" || strings.TrimSpace(*batchOut) == "" {
			log.Fatalf("both --batch-seeds and --batch-out are required in batch mode")
		}
		seedFrom, seedTo, err := ParseSeedRange(*batchSeeds)
		if err != nil {
			log.Fatalf("invalid --batch-seeds: %v", err)
		}
		runs, err := RunSeedBatch(SeedBatchConfig{
			SeedFrom: seedFrom,
			SeedTo:   seedTo,
			Ticks:    *batchTicks,
			Output:   *batchOut,
		})
		if err != nil {
			log.Fatalf("batch failed: %v", err)
		}
		fmt.Printf("✅ Batch completed: runs=%d seeds=%d:%d ticks=%d csv=%s\n", runs, seedFrom, seedTo, *batchTicks, *batchOut)
		return
	}

	simCfg := SimulationConfig{}
	if *deterministic {
		simCfg.Deterministic = true
		simCfg.Seed = *seed
	}

	// Initialize simulator (the heart of the backend)
	simulator := NewWorldSimulatorWithConfig(simCfg)
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
