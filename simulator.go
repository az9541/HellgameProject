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
	GlobalTick int64
	// Event tracking
	EventLog []WorldEvent
	mu       sync.RWMutex
	// Channels for goroutines
	stop     chan bool
	Wars     map[string]*WarState
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

// WorldEvent представляет событие в мире
type WorldEvent struct {
	ID          string
	Tick        int64
	Type        string // "faction_war", "trade_route", "rebellion", "discovery"
	Location    string // domain ID
	Title       string
	Description string
	Consequence string
	Factions    []string // involved factions
}

// SimulationDelta - результат симуляции
type SimulationDelta struct {
	TicksSimulated int64
	Events         []WorldEvent
	FactionStates  map[string]*FactionState
	DomainStates   map[string]*DomainState
	GlobalTick     int64
}

// NewWorldSimulator создаёт новый симулятор
func NewWorldSimulator() *WorldSimulator {
	domains, _ := createInitialDomains()
	sim := &WorldSimulator{
		Factions:   createInitialFactions(),
		Domains:    domains,
		GlobalTick: 0,
		EventLog:   []WorldEvent{},
		stop:       make(chan bool),
		EventBus:   NewEventPublisher(),
	}
	sim.initializeFactionInfluence()
	sim.Wars = make(map[string]*WarState)
	return sim
}

// Start запускает фоновые горутины симуляции
func (sim *WorldSimulator) Start() {
	log.Println("🚀 Starting world simulation goroutines...")

	go sim.runTimeLoop()

	log.Printf("✅ Simulation goroutines started")
}

// Stop останавливает симуляцию
func (sim *WorldSimulator) Stop() {
	sim.stop <- true
	log.Println("Simulation stopped")
}

// Simulate запускает симуляцию на N часов
func (sim *WorldSimulator) Simulate(ticks int64) *SimulationDelta {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	startTime := sim.GlobalTick
	endTime := startTime + ticks

	// Отслеживаем события, проходящие во время симуляции
	newEvents := []WorldEvent{}

	for tick := startTime; tick < endTime; tick++ {
		sim.GlobalTick = tick

		// Каждый игровой час есть фиксированное значение вероятности наступления события
		if rand.Float64() < 0.3 { // 30% chance per tick
			event := sim.generateTickEvent(tick)
			if event != nil {
				sim.EventLog = append(sim.EventLog, *event)
				newEvents = append(newEvents, *event)
			}
		}

		sim.executeFactionActions()

		sim.runKPPSimulation()

		// Синхронизируем списки доменов у фракций
		sim.syncFactionDomains()

		sim.updateDomainStability()
		sim.UpdateWars()
	}

	// Return delta (only changes)
	delta := &SimulationDelta{
		TicksSimulated: ticks,
		Events:         newEvents,
		FactionStates:  sim.copyFactionStates(),
		DomainStates:   sim.copyDomainStates(),
		GlobalTick:     sim.GlobalTick,
	}

	log.Printf("SIMULATE_COMPLETE ticks=%d global_tick=%d new_events=%d", ticks, sim.GlobalTick, len(newEvents))
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
			sim.mu.Lock()

			// 1. Сначала считаем физику (влияние распространяется)
			sim.runKPPSimulation()

			// 2. Раз в 6 тиков (30 сек) фракции принимают решения
			if sim.GlobalTick%6 == 0 {
				sim.executeFactionActions()
			}

			// 3. Раз в 9 тиков (45 сек) происходят случайные события
			if sim.GlobalTick%9 == 0 {
				event := sim.generateTickEvent(sim.GlobalTick)
				if event != nil {
					sim.EventLog = append(sim.EventLog, *event)
				}
			}

			// 4. Раз в 12 тиков (60 сек) обновляем стабильность доменов
			if sim.GlobalTick%12 == 0 {
				sim.updateDomainStability()
			}

			sim.updateFactionMilitaryForce()
			sim.UpdateWars()

			// 5. И только в конце обновляем время
			sim.GlobalTick++

			sim.mu.Unlock()
		}
	}
}
