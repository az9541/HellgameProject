package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

// WorldSimulator управляет симуляцией всего мира
type WorldSimulator struct {
	// State
	Factions   map[string]*FactionState
	Domains    map[string]*DomainState
	GlobalTime int64

	// Event tracking
	EventLog []WorldEvent
	mu       sync.RWMutex

	// Channels for goroutines
	stop chan bool
}

// FactionState отслеживает состояние фракции
type FactionState struct {
	ID             string
	Name           string
	Power          float64            // 0-100
	Territory      float64            // total size
	DomainsHeld    []string           // domain IDs
	Attitude       map[string]float64 // towards other factions: -100 to +100
	Resources      float64            // wealth/supplies
	MilitaryForce  float64            // strength 0-100
	LastActionTime int64
}

// DomainState отслеживает состояние домена
type DomainState struct {
	ID           string
	Name         string
	Stability    float64 // 0-100
	ControlledBy string  // faction ID
	DangerLevel  int     // 1-10
	Population   int
	Mood         string   // "stable", "unrest", "rebellion"
	Events       []string // event IDs that happened here
}

// WorldEvent представляет событие в мире
type WorldEvent struct {
	ID          string
	Hour        int64
	Type        string // "faction_war", "trade_route", "rebellion", "discovery"
	Location    string // domain ID
	Title       string
	Description string
	Consequence string
	Factions    []string // involved factions
}

// SimulationDelta - результат симуляции
type SimulationDelta struct {
	HoursSimulated int64
	Events         []WorldEvent
	FactionStates  map[string]*FactionState
	DomainStates   map[string]*DomainState
	GlobalTime     int64
}

// NewWorldSimulator создаёт новый симулятор
func NewWorldSimulator() *WorldSimulator {
	sim := &WorldSimulator{
		Factions:   createInitialFactions(),
		Domains:    createInitialDomains(),
		GlobalTime: 0,
		EventLog:   []WorldEvent{},
		stop:       make(chan bool),
	}
	return sim
}

// Start запускает фоновые горутины симуляции
func (sim *WorldSimulator) Start() {
	log.Println("🚀 Starting world simulation goroutines...")

	// Фракции воюют и торгуют
	go sim.runFactionAI()

	// Домены меняют стабильность
	go sim.runDomainSimulation()

	// Случайные события
	go sim.runEventGenerator()

	log.Println("✅ Simulation goroutines started")
}

// Stop останавливает симуляцию
func (sim *WorldSimulator) Stop() {
	sim.stop <- true
	log.Println("⛔ Simulation stopped")
}

// ============ MAIN SIMULATION LOOP ============

// Simulate запускает симуляцию на N часов
func (sim *WorldSimulator) Simulate(hours int64) *SimulationDelta {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	startTime := sim.GlobalTime
	endTime := startTime + hours

	// Track events that happen during this simulation
	newEvents := []WorldEvent{}

	for hour := startTime; hour < endTime; hour++ {
		sim.GlobalTime = hour

		// Every hour: chance of events
		if rand.Float64() < 0.3 { // 30% chance per hour
			event := sim.generateHourlyEvent(hour)
			if event != nil {
				sim.EventLog = append(sim.EventLog, *event)
				newEvents = append(newEvents, *event)
			}
		}

		// Every 12 hours: faction actions
		if hour%12 == 0 {
			sim.executeFactionActions()
		}

		// Every 6 hours: domain stability changes
		if hour%6 == 0 {
			sim.updateDomainStability()
		}
	}

	// Return delta (only changes)
	delta := &SimulationDelta{
		HoursSimulated: hours,
		Events:         newEvents,
		FactionStates:  sim.copyFactionStates(),
		DomainStates:   sim.copyDomainStates(),
		GlobalTime:     sim.GlobalTime,
	}

	log.Printf("✅ Simulated %d hours (time: %d)", hours, sim.GlobalTime)
	return delta
}

// ============ GOROUTINE: FACTION AI ============

func (sim *WorldSimulator) runFactionAI() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-sim.stop:
			return
		case <-ticker.C:
			sim.mu.Lock()
			sim.executeFactionActions()
			sim.mu.Unlock()
		}
	}
}

func (sim *WorldSimulator) executeFactionActions() {
	for _, faction := range sim.Factions {
		// Random chance: faction does something
		if rand.Float64() < 0.4 { // 40% chance per tick
			action := rand.Intn(3)
			switch action {
			case 0:
				// Try to take control of a domain
				sim.attemptDomainTakeover(faction)
			case 1:
				// Establish trade route
				sim.establishTradeRoute(faction)
			case 2:
				// Recruit resources
				faction.Resources = minFloat(faction.Resources+5, 100)
			}
		}
	}
}

func (sim *WorldSimulator) attemptDomainTakeover(attacker *FactionState) {
	// Find a domain not controlled by this faction
	for _, domain := range sim.Domains {
		if domain.ControlledBy == attacker.ID {
			continue // Already controls it
		}

		// Check probability based on military force vs domain danger
		probability := (attacker.MilitaryForce / 100) * (1 - float64(domain.DangerLevel)/10)

		if rand.Float64() < probability {
			defender := sim.Factions[domain.ControlledBy]
			if defender == nil {
				// Domain is uncontrolled, take it
				domain.ControlledBy = attacker.ID
				attacker.DomainsHeld = append(attacker.DomainsHeld, domain.ID)
				attacker.Power += 5
				log.Printf("🎖️  %s takes control of %s", attacker.Name, domain.Name)
			} else {
				// War between factions
				outcome := sim.resolveFactionWar(attacker, defender, domain)
				log.Printf("⚔️  War: %s vs %s in %s → %s", attacker.Name, defender.Name, domain.Name, outcome)
			}
		}
	}
}

func (sim *WorldSimulator) resolveFactionWar(attacker, defender *FactionState, domain *DomainState) string {
	// Compare military forces
	attackerStrength := attacker.MilitaryForce * (1 + rand.Float64()*0.2) // ±20% variance
	defenderStrength := defender.MilitaryForce * (1 + rand.Float64()*0.2)

	if attackerStrength > defenderStrength {
		// Attacker wins
		// Transfer domain
		domain.ControlledBy = attacker.ID
		attacker.DomainsHeld = append(attacker.DomainsHeld, domain.ID)

		// Remove from defender
		newDomains := []string{}
		for _, d := range defender.DomainsHeld {
			if d != domain.ID {
				newDomains = append(newDomains, d)
			}
		}
		defender.DomainsHeld = newDomains

		// Adjust power
		attacker.Power += 8
		defender.Power -= 5

		// Stability drops in conquered domain
		domain.Stability -= 15
		domain.Mood = "unrest"

		return "attacker_victory"
	} else {
		// Defender wins
		defender.Power += 5
		attacker.Power -= 3
		return "defender_victory"
	}
}

func (sim *WorldSimulator) establishTradeRoute(faction *FactionState) {
	// Find two domains
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		domains = append(domains, d)
	}

	if len(domains) < 2 {
		return
	}

	domain1 := domains[rand.Intn(len(domains))]
	domain2 := domains[rand.Intn(len(domains))]

	if domain1.ID == domain2.ID {
		return
	}

	// Establish trade (improve stability in both domains)
	domain1.Stability = minFloat(domain1.Stability+10, 100)
	domain2.Stability = minFloat(domain2.Stability+10, 100)
	faction.Resources += 10

	log.Printf("🏪 Trade route established between %s and %s by %s", domain1.Name, domain2.Name, faction.Name)
}

// ============ GOROUTINE: DOMAIN SIMULATION ============

func (sim *WorldSimulator) runDomainSimulation() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sim.stop:
			return
		case <-ticker.C:
			sim.mu.Lock()
			sim.updateDomainStability()
			sim.mu.Unlock()
		}
	}
}

func (sim *WorldSimulator) updateDomainStability() {
	for _, domain := range sim.Domains {
		controller := sim.Factions[domain.ControlledBy]
		if controller == nil {
			domain.Stability = maxFloat(domain.Stability-2, 0) // Ungoverned → chaos
			continue
		}

		// Stability based on faction's ideology and power
		if controller.ID == FactionCorporateConsortium {
			// Corporate = stable but exploitative
			domain.Stability = minFloat(domain.Stability+1, 80)
		} else if controller.ID == FactionRepentantCommunes {
			// Communes = moderate stability, good morale
			domain.Stability = minFloat(domain.Stability+2, 90)
		} else if controller.ID == FactionNeoTormentors {
			// Neo-Tormentors = oppressive but effective
			domain.Stability = minFloat(domain.Stability+0.5, 70)
		}

		// Danger level decreases with stability
		if domain.Stability > 70 {
			domain.DangerLevel = maxInt(domain.DangerLevel-1, 1)
		} else if domain.Stability < 30 {
			domain.DangerLevel = minInt(domain.DangerLevel+1, 10)
		}
	}
}

// ============ GOROUTINE: EVENT GENERATION ============

func (sim *WorldSimulator) runEventGenerator() {
	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sim.stop:
			return
		case <-ticker.C:
			sim.mu.Lock()
			if event := sim.generateHourlyEvent(sim.GlobalTime); event != nil {
				sim.EventLog = append(sim.EventLog, *event)
			}
			sim.mu.Unlock()
		}
	}
}

func (sim *WorldSimulator) generateHourlyEvent(hour int64) *WorldEvent {
	eventType := rand.Intn(5)

	switch eventType {
	case 0:
		return sim.generateMysteryEvent(hour)
	case 1:
		return sim.generateResourceEvent(hour)
	case 2:
		return sim.generateCulturalEvent(hour)
	case 3:
		return sim.generateDangerEvent(hour)
	default:
		return nil
	}
}

func (sim *WorldSimulator) generateMysteryEvent(hour int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		domains = append(domains, d)
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	titles := []string{
		"Ancient entity stirs in the shadows",
		"A mysterious figure appears in the mist",
		"Strange markings discovered on ancient stones",
		"Whispers of something forgotten echo through the domain",
	}

	return &WorldEvent{
		ID:          generateID(),
		Hour:        hour,
		Type:        "mystery",
		Location:    domain.ID,
		Title:       titles[rand.Intn(len(titles))],
		Description: "Something ancient and unknown has stirred...",
		Consequence: "heresy_danger_level +2",
	}
}

func (sim *WorldSimulator) generateResourceEvent(hour int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionCorporateConsortium {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        hour,
		Type:        "resource_discovery",
		Location:    domain.ID,
		Title:       "New mineral deposits discovered",
		Description: "Corporate teams have found rich deposits of infernal ore.",
		Consequence: "corporate_consortium power +3",
	}
}

func (sim *WorldSimulator) generateCulturalEvent(hour int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionRepentantCommunes {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        hour,
		Type:        "cultural",
		Location:    domain.ID,
		Title:       "Community gathering brings hope",
		Description: "The communes organize a gathering to celebrate survival and mutual aid.",
		Consequence: domain.Name + " stability +5",
	}
}

func (sim *WorldSimulator) generateDangerEvent(hour int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.DangerLevel > 5 {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        hour,
		Type:        "danger",
		Location:    domain.ID,
		Title:       "Dangerous creature sighting",
		Description: "Reports of a dangerous entity roaming the domain.",
		Consequence: domain.Name + " danger_level +1",
	}
}

// ============ HELPERS ============

func (sim *WorldSimulator) copyFactionStates() map[string]*FactionState {
	copy := make(map[string]*FactionState)
	for id, faction := range sim.Factions {
		copy[id] = &FactionState{
			ID:            faction.ID,
			Name:          faction.Name,
			Power:         faction.Power,
			Territory:     faction.Territory,
			DomainsHeld:   append([]string{}, faction.DomainsHeld...),
			Attitude:      faction.Attitude,
			Resources:     faction.Resources,
			MilitaryForce: faction.MilitaryForce,
		}
	}
	return copy
}

func (sim *WorldSimulator) copyDomainStates() map[string]*DomainState {
	copy := make(map[string]*DomainState)
	for id, domain := range sim.Domains {
		copy[id] = &DomainState{
			ID:           domain.ID,
			Name:         domain.Name,
			Stability:    domain.Stability,
			ControlledBy: domain.ControlledBy,
			DangerLevel:  domain.DangerLevel,
			Population:   domain.Population,
			Mood:         domain.Mood,
		}
	}
	return copy
}

func createInitialFactions() map[string]*FactionState {
	return map[string]*FactionState{
		FactionCorporateConsortium: {
			ID:             FactionCorporateConsortium,
			Name:           "Corporate Consortium",
			Power:          70,
			Territory:      3.5,
			DomainsHeld:    []string{DomainLust, DomainGreed},
			Attitude:       make(map[string]float64),
			Resources:      80,
			MilitaryForce:  75,
			LastActionTime: 0,
		},
		FactionRepentantCommunes: {
			ID:             FactionRepentantCommunes,
			Name:           "Repentant Communes",
			Power:          50,
			Territory:      1.8,
			DomainsHeld:    []string{DomainGluttony},
			Attitude:       make(map[string]float64),
			Resources:      40,
			MilitaryForce:  35,
			LastActionTime: 0,
		},
		FactionNeoTormentors: {
			ID:             FactionNeoTormentors,
			Name:           "Neo-Tormentors",
			Power:          65,
			Territory:      2.5,
			DomainsHeld:    []string{DomainWrath, DomainViolence},
			Attitude:       make(map[string]float64),
			Resources:      70,
			MilitaryForce:  85,
			LastActionTime: 0,
		},
		FactionCaravanGuilds: {
			ID:             FactionCaravanGuilds,
			Name:           "Caravan Guilds",
			Power:          45,
			Territory:      0.9,
			DomainsHeld:    []string{DomainLimbo},
			Attitude:       make(map[string]float64),
			Resources:      60,
			MilitaryForce:  40,
			LastActionTime: 0,
		},
		FactionAncientRemnants: {
			ID:             FactionAncientRemnants,
			Name:           "Ancient Remnants",
			Power:          30,
			Territory:      0.3,
			DomainsHeld:    []string{DomainHeresy},
			Attitude:       make(map[string]float64),
			Resources:      50,
			MilitaryForce:  60,
			LastActionTime: 0,
		},
	}
}

func createInitialDomains() map[string]*DomainState {
	return map[string]*DomainState{
		DomainLimbo: {
			ID:           DomainLimbo,
			Name:         "Limbo",
			Stability:    60,
			ControlledBy: FactionCaravanGuilds,
			DangerLevel:  3,
			Population:   5000,
			Mood:         "stable",
		},
		DomainLust: {
			ID:           DomainLust,
			Name:         "Circle of Lust",
			Stability:    40,
			ControlledBy: FactionCorporateConsortium,
			DangerLevel:  5,
			Population:   3000,
			Mood:         "exploited",
		},
		DomainGluttony: {
			ID:           DomainGluttony,
			Name:         "Circle of Gluttony",
			Stability:    50,
			ControlledBy: FactionRepentantCommunes,
			DangerLevel:  4,
			Population:   2500,
			Mood:         "hopeful",
		},
		DomainGreed: {
			ID:           DomainGreed,
			Name:         "Circle of Greed",
			Stability:    35,
			ControlledBy: FactionCorporateConsortium,
			DangerLevel:  6,
			Population:   4000,
			Mood:         "unrest",
		},
		DomainWrath: {
			ID:           DomainWrath,
			Name:         "Circle of Wrath",
			Stability:    20,
			ControlledBy: FactionNeoTormentors,
			DangerLevel:  9,
			Population:   6000,
			Mood:         "terrified",
		},
		DomainHeresy: {
			ID:           DomainHeresy,
			Name:         "Circle of Heresy",
			Stability:    45,
			ControlledBy: FactionAncientRemnants,
			DangerLevel:  7,
			Population:   1000,
			Mood:         "mysterious",
		},
		DomainViolence: {
			ID:           DomainViolence,
			Name:         "Circle of Violence",
			Stability:    15,
			ControlledBy: FactionNeoTormentors,
			DangerLevel:  10,
			Population:   8000,
			Mood:         "chaotic",
		},
		DomainFraud: {
			ID:           DomainFraud,
			Name:         "Circle of Fraud",
			Stability:    30,
			ControlledBy: "none",
			DangerLevel:  8,
			Population:   2000,
			Mood:         "deceptive",
		},
		DomainBetrayance: {
			ID:           DomainBetrayance,
			Name:         "Ninth Circle",
			Stability:    10,
			ControlledBy: "none",
			DangerLevel:  10,
			Population:   500,
			Mood:         "despairing",
		},
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func generateID() string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	id := "evt_"
	for i := 0; i < 3; i++ {
		id += string(chars[rand.Intn(len(chars))])
	}
	return id
}
