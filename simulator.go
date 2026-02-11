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
	// Event tracking
	eventMu sync.RWMutex
	mu      sync.RWMutex
	// Channels for goroutines
	stop     chan bool
	EventBus *EventPublisher
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
	ID              string
	Name            string
	Stability       float64 // 0-100
	ControlledBy    string  // faction ID
	DangerLevel     int     // 1-10
	Population      int
	Mood            string   // "stable", "unrest", "rebellion"
	Events          []string // event IDs that happened here
	Influence       map[string]float64
	AdjacentDomains []string // Neighbours to domain
}

type WarState struct {
	ID             string // "war:<domainID>:<attackerID>:<defenderID>"
	AttackerID     string
	DefenderID     string
	DomainID       string
	StartTick      int64
	LastUpdateTick int64
	TicksDuration  int64
	// Замороженные плотности влияния на момент начала войны
	FrozenFactionsDenseties map[string]float64
	// Выделенные контингенты на войну (фиксируются при старте)
	AttackerCommittedForce float64 // начальный контингент атакующего
	DefenderCommittedForce float64 // начальный контингент защитника
	AttackerCurrentForce   float64 // текущие силы контингента атакующего
	DefenderCurrentForce   float64 // текущие силы контингента защитника
	// Динамика войны
	Momentum       float64 // положительное — преимущество атакующего
	AttackerMorale float64 // [0,100]
	DefenderMorale float64 // [0,100]
	// Итоги
	IsOver    bool
	WinnersID map[string]string
	LosersID  map[string]string
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
			sim.EmitEvent(*event)
		}
	}
	// 4. Раз в 12 тиков (60 сек) обновляем стабильность доменов
	if sim.State.GlobalTick%12 == 0 {
		sim.updateDomainStability()
	}
	sim.updateFactionMilitaryForce()
	sim.UpdateWars()

	// 5. И только в конце обновляем время
	sim.State.GlobalTick++
}

// Simulate запускает симуляцию на N часов
func (sim *WorldSimulator) Simulate(ticks int64) *SimulationDelta {
	startTime := sim.State.GlobalTick
	endTime := startTime + ticks

	for tick := startTime; tick < endTime; tick++ {
		sim.Tick()
	}

	// Return delta (only changes)
	delta := &SimulationDelta{
		TicksSimulated: ticks,
		Events:         sim.State.EventLog,
		FactionStates:  sim.copyFactionStates(),
		DomainStates:   sim.copyDomainStates(),
		GlobalTick:     sim.State.GlobalTick,
	}

	sim.EmitEvent(GameEvent{
		Type:      "SIMULATION_COMPLETED",
		Tick:      sim.State.GlobalTick,
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
	ticker := time.NewTicker(5 * time.Second)
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
