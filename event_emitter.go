package main

// EmitEvent - это основной метод для генерации событий в мире.
// Он принимает GameEvent, который содержит всю необходимую информацию о событии,
// и возвращает WorldEvent, который может быть сохранён в журнале событий и отображён в UI.
// Нужен для того, чтобы обеспечить единый интерфейс для генерации событий
// , который может быть использован в любом месте симулятора мира.

func (sim *WorldSimulator) EmitEvent(event GameEvent) WorldEvent {
	sim.eventMu.Lock()
	worldEvent := sim.emitEventLocked(event) // только лог
	sim.eventMu.Unlock()

	if sim.EventBus != nil {
		sim.EventBus.Publish(event)
	}
	return worldEvent
}

// emitEventLocked - внутренний метод для генерации событий, который должен вызываться с уже заблокированным eventMu.
// Нужен для того, чтобы избежать двойного блокирования,
// если EmitEvent вызывается изнутри других методов, которые уже держат блокировку.
// emitEventLocked делает ТОЛЬКО работу с EventLog и метаданными
func (sim *WorldSimulator) emitEventLocked(event GameEvent) WorldEvent {
	worldMeta := WorldEventMeta{}
	if carrier, ok := event.EventData.(WorldEventMetaProvider); ok {
		if meta := carrier.EventMeta(); meta != nil {
			worldMeta = *meta
		}
	}
	if worldMeta.ID == "" {
		worldMeta.ID = generateID()
	}

	worldEvent := WorldEvent{
		ID:          worldMeta.ID,
		Tick:        event.Tick,
		Type:        event.Type,
		Location:    worldMeta.Location,
		Title:       worldMeta.Title,
		Description: worldMeta.Description,
		Consequence: worldMeta.Consequence,
		Factions:    append([]string{}, worldMeta.Factions...),
	}

	sim.EventLog = append(sim.EventLog, worldEvent)
	return worldEvent
}
