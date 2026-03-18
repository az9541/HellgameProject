package engine

// GameEngine - интерфейс ядра
// Описывает, что в принципе может делать ядро, без привязки к конкретной реализации (например, WorldSimulator).
type GameEngine interface {
	Simulate(ticks int64) *SimulationDelta // Ядро может запускать симуляцию на N часов и возвращать изменения (delta)
	GetWorldState() *WorldStateSnapshot    // Ядро может возвращать текущее состояние мира
	GetEvents(limit int) []GameEvent       // Ядро может возвращать события из лога (с лимитом)
	GetFactions() map[string]*FactionState // Ядро может возвращать состояние всех фракций
	GetDomains() map[string]*DomainState   // Ядро может возвращать состояние всех доменов
}

type WorldStateSnapshot struct {
	Time     int64
	Factions map[string]*FactionState
	Domains  map[string]*DomainState
	Wars     map[string]*WarState
	EventLog []GameEvent
}
