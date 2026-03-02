package engine

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type SeedBatchConfig struct {
	SeedFrom int
	SeedTo   int
	Ticks    int64
	Output   string
}

func (cfg SeedBatchConfig) validate() error {
	if cfg.SeedFrom <= 0 || cfg.SeedTo <= 0 {
		return fmt.Errorf("seed range must be positive: got %d:%d", cfg.SeedFrom, cfg.SeedTo)
	}
	if cfg.SeedTo < cfg.SeedFrom {
		return fmt.Errorf("seed range must be ascending: got %d:%d", cfg.SeedFrom, cfg.SeedTo)
	}
	if cfg.Ticks <= 0 {
		return fmt.Errorf("ticks must be > 0: got %d", cfg.Ticks)
	}
	if strings.TrimSpace(cfg.Output) == "" {
		return fmt.Errorf("output path is required")
	}
	return nil
}

func ParseSeedRange(raw string) (int, int, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, 0, fmt.Errorf("empty seed range")
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		v, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid seed %q: %w", s, err)
		}
		return v, v, nil
	case 2:
		from, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid seed range start %q: %w", parts[0], err)
		}
		to, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid seed range end %q: %w", parts[1], err)
		}
		return from, to, nil
	default:
		return 0, 0, fmt.Errorf("invalid seed range format %q, expected N or A:B", s)
	}
}

func RunSeedBatch(cfg SeedBatchConfig) (int, error) {
	if err := cfg.validate(); err != nil {
		return 0, err
	}

	outputDir := filepath.Dir(cfg.Output)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return 0, fmt.Errorf("create output dir %q: %w", outputDir, err)
		}
	}

	file, err := os.Create(cfg.Output)
	if err != nil {
		return 0, fmt.Errorf("create output file %q: %w", cfg.Output, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	factionIDs := getSortedFactionKeys(createInitialFactions())
	header := []string{
		"seed",
		"ticks",
		"end_tick",
		"total_events",
		"total_wars",
		"active_wars",
		"extinct_count",
		"extinct_factions",
		"dominant_faction",
		"dominant_domains",
	}
	for _, factionID := range factionIDs {
		header = append(header,
			factionID+"_domains",
			factionID+"_military",
			factionID+"_resources",
			factionID+"_power",
		)
	}

	if err := writer.Write(header); err != nil {
		return 0, fmt.Errorf("write csv header: %w", err)
	}

	runs := 0
	for seed := cfg.SeedFrom; seed <= cfg.SeedTo; seed++ {
		sim := NewWorldSimulatorWithConfig(SimulationConfig{
			Deterministic: true,
			Seed:          int64(seed),
		})

		sim.Simulate(cfg.Ticks)

		domainCount := make(map[string]int, len(factionIDs))
		for _, factionID := range factionIDs {
			domainCount[factionID] = 0
		}
		for _, domain := range sim.State.Domains {
			if _, ok := domainCount[domain.ControlledBy]; ok {
				domainCount[domain.ControlledBy]++
			}
		}

		extinct := make([]string, 0, len(factionIDs))
		dominantFaction := ""
		dominantDomains := -1
		for _, factionID := range factionIDs {
			count := domainCount[factionID]
			if count == 0 {
				extinct = append(extinct, factionID)
			}
			if count > dominantDomains || (count == dominantDomains && (dominantFaction == "" || factionID < dominantFaction)) {
				dominantDomains = count
				dominantFaction = factionID
			}
		}

		activeWars := 0
		for _, war := range sim.State.Wars {
			if war != nil && !war.IsOver {
				activeWars++
			}
		}

		row := []string{
			strconv.Itoa(seed),
			strconv.FormatInt(cfg.Ticks, 10),
			strconv.FormatInt(sim.State.GlobalTick, 10),
			strconv.Itoa(len(sim.State.EventLog)),
			strconv.Itoa(len(sim.State.Wars)),
			strconv.Itoa(activeWars),
			strconv.Itoa(len(extinct)),
			strings.Join(extinct, "|"),
			dominantFaction,
			strconv.Itoa(dominantDomains),
		}

		for _, factionID := range factionIDs {
			faction := sim.State.Factions[factionID]
			military := 0.0
			resources := 0.0
			power := 0.0
			if faction != nil {
				military = faction.MilitaryForce
				resources = faction.Resources
				power = faction.Power
			}

			row = append(row,
				strconv.Itoa(domainCount[factionID]),
				strconv.FormatFloat(military, 'f', 4, 64),
				strconv.FormatFloat(resources, 'f', 4, 64),
				strconv.FormatFloat(power, 'f', 4, 64),
			)
		}

		if err := writer.Write(row); err != nil {
			return runs, fmt.Errorf("write csv row for seed %d: %w", seed, err)
		}
		runs++
	}

	if err := writer.Error(); err != nil {
		return runs, fmt.Errorf("flush csv writer: %w", err)
	}

	return runs, nil
}
