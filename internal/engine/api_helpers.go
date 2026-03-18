package engine

// api_helpers.go - содержит адаптеры и вспомогательные функции для API-слоя, которые используют GameEngine интерфейс.
// Эти функции позволяют API-обработчикам работать с ядром (WorldSimulator) через абстрактный интерфейс (GameEngine).
// Это позволяет не зависеть от конкретной реализации (хотя оно и будет в единственном экземпляре).
// И самое главное, API-слои пересают вмешиваться во внутреннюю логику симулятора, а работают через четко определенные методы интерфейса.

func (sim *WorldSimulator) Simulate(ticks int64) *SimulationDelta {
	sim.Mu.RLock()
	startTime := sim.State.GlobalTick
	sim.Mu.RUnlock()
	endTime := startTime + ticks

	for tick := startTime; tick < endTime; tick++ {
		sim.Tick()
	}

	// Return delta (only changes)
	sim.Mu.RLock()
	delta := &SimulationDelta{
		TicksSimulated: ticks,
		Events:         sim.copyEventLog(),
		FactionStates:  sim.CopyFactionStates(),
		DomainStates:   sim.CopyDomainStates(),
		GlobalTick:     sim.State.GlobalTick,
	}
	completionTick := sim.State.GlobalTick
	sim.Mu.RUnlock()

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

func (sim *WorldSimulator) GetWorldState() *WorldStateSnapshot {
	sim.Mu.RLock()
	defer sim.Mu.RUnlock()

	result := &WorldStateSnapshot{
		Time:     sim.State.GlobalTick,
		Factions: sim.CopyFactionStates(),
		Domains:  sim.CopyDomainStates(),
		Wars:     sim.CopyWars(),
		EventLog: sim.copyEventLog(),
	}
	return result
}

func (sim *WorldSimulator) GetEvents(limit int) []GameEvent {
	sim.Mu.RLock()
	defer sim.Mu.RUnlock()

	events := sim.State.EventLog

	// Return last N events
	start := len(events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]GameEvent, len(events)-start)
	copy(result, events[start:])
	return result
}

func (sim *WorldSimulator) GetFactions() map[string]*FactionState {
	sim.Mu.RLock()
	defer sim.Mu.RUnlock()
	return sim.CopyFactionStates()
}

func (sim *WorldSimulator) GetDomains() map[string]*DomainState {
	sim.Mu.RLock()
	defer sim.Mu.RUnlock()
	return sim.CopyDomainStates()
}
