package main

import (
	"log"
	"sync"
	"time"
)

// WorldSimulator управляет симуляцией всего мира
type WorldSimulator struct {
	// State
	State *WorldState
	mu    sync.RWMutex
	// Channels for goroutines
	stop     chan bool
	EventBus *EventPublisher
}

// FactionState отслеживает состояние фракции
type FactionState struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Power          float64            `json:"power"`
	Territory      float64            `json:"territory"`
	DomainsHeld    []string           `json:"domains_held"`
	Attitude       map[string]float64 `json:"attitude"`
	Resources      float64            `json:"resources"`
	MilitaryForce  float64            `json:"military_force"`
	LastActionTime int64              `json:"last_action_time"`
}

// DomainState отслеживает состояние домена
type DomainState struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Stability       float64            `json:"stability"`
	ControlledBy    string             `json:"controlled_by"`
	DangerLevel     float64            `json:"danger_level"`
	Population      int                `json:"population"`
	Mood            string             `json:"mood"`
	Events          []string           `json:"events"`
	Influence       map[string]float64 `json:"influence"`
	AdjacentDomains []string           `json:"adjacent_domains"`
	Resources       float64            `json:"resources"`
}

type WarState struct {
	ID                      string             `json:"id"`
	AttackerID              string             `json:"attacker_id"`
	DefenderID              string             `json:"defender_id"`
	DomainID                string             `json:"domain_id"`
	StartTick               int64              `json:"start_tick"`
	LastUpdateTick          int64              `json:"last_update_tick"`
	TicksDuration           int64              `json:"ticks_duration"`
	FrozenFactionsDenseties map[string]float64 `json:"frozen_factions_denseties"`
	AttackerCommittedForce  float64            `json:"attacker_committed_force"`
	DefenderCommittedForce  float64            `json:"defender_committed_force"`
	AttackerCurrentForce    float64            `json:"attacker_current_force"`
	DefenderCurrentForce    float64            `json:"defender_current_force"`
	Momentum                float64            `json:"momentum"`
	AttackerMorale          float64            `json:"attacker_morale"`
	DefenderMorale          float64            `json:"defender_morale"`
	IsOver                  bool               `json:"is_over"`
	WinnersID               map[string]string  `json:"winners_id"`
	LosersID                map[string]string  `json:"losers_id"`
}

// SimulationDelta - результат симуляции
type SimulationDelta struct {
	TicksSimulated int64
	Events         []GameEvent
	FactionStates  map[string]*FactionState
	DomainStates   map[string]*DomainState
	GlobalTick     int64
}

// Полное состояние мира. По сути является снапшотом текущего тика.
// Должно использоваться в передаче в Godot по API, а также для сохранения/загрузки игры.
type WorldState struct {
	Factions   map[string]*FactionState
	Domains    map[string]*DomainState
	Wars       map[string]*WarState
	GlobalTick int64
	EventLog   []GameEvent
}

// NewWorldSimulator создаёт новый симулятор
func NewWorldSimulator() *WorldSimulator {
	domains, _ := createInitialDomains()
	state := &WorldState{
		Factions:   createInitialFactions(),
		Domains:    domains,
		Wars:       make(map[string]*WarState),
		GlobalTick: 0,
		EventLog:   make([]GameEvent, 0),
	}

	sim := &WorldSimulator{
		State:    state,
		stop:     make(chan bool),
		EventBus: NewEventPublisher(),
	}
	sim.initializeFactionInfluence()
	return sim
}

// Start запускает фоновые горутины симуляции
func (sim *WorldSimulator) Start() {
	log.Println("🚀 Starting world simulation goroutines...")

	go sim.runTimeLoop()
	//go sim.Simulate(2000)

	log.Printf("✅ Simulation goroutines started")
}

// Stop останавливает симуляцию
func (sim *WorldSimulator) Stop() {
	sim.stop <- true
	log.Println("Simulation stopped")
}

func (sim *WorldSimulator) Tick() {
	defer sim.mu.Unlock()
	sim.mu.Lock()
	sim.runKPPSimulation()
	if sim.State.GlobalTick%6 == 0 {
		sim.executeFactionActions()
	}
	// 3. Раз в 9 тиков (45 сек) происходят случайные события
	if sim.State.GlobalTick%9 == 0 {
		event := sim.generateTickEvent(sim.State.GlobalTick)
		if event != nil {
			sim.emitEventLocked(*event)
		}
	}
	sim.UpdateDomainStability()
	sim.UpdateDomainDanger()
	sim.UpdateFactionMilitaryForce()
	sim.UpdateFactionsOtherParameters()
	sim.UpdateDomainResources()
	sim.UpdateWars()

	// 5. И только в конце обновляем время
	sim.State.GlobalTick++
}

// Simulate запускает симуляцию на N часов
func (sim *WorldSimulator) Simulate(ticks int64) *SimulationDelta {
	sim.mu.RLock()
	startTime := sim.State.GlobalTick
	sim.mu.RUnlock()
	endTime := startTime + ticks

	for tick := startTime; tick < endTime; tick++ {
		sim.Tick()
	}

	// Return delta (only changes)
	sim.mu.RLock()
	delta := &SimulationDelta{
		TicksSimulated: ticks,
		Events:         sim.copyEventLog(),
		FactionStates:  sim.copyFactionStates(),
		DomainStates:   sim.copyDomainStates(),
		GlobalTick:     sim.State.GlobalTick,
	}
	completionTick := sim.State.GlobalTick
	sim.mu.RUnlock()

	sim.EmitEvent(GameEvent{
		Type:      "SIMULATION_COMPLETED",
		Tick:      completionTick,
		EventKind: EventKindGeneric,
		EventData: GenericEventData{
			EventKind: EventKindGeneric,
			EventData: map[string]any{
				"ticks_simulated": ticks,
				"events_count":    len(delta.Events),
				"factions":        delta.FactionStates,
				"domains":         delta.DomainStates,
			},
		},
	})
	return delta
}

// runTimeLoop - главный цикл фоновой симуляции
func (sim *WorldSimulator) runTimeLoop() {
	// 1 тик = 5 реальных секунд
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-sim.stop:
			return
		case <-ticker.C:
			sim.Tick()
		}
	}
}
