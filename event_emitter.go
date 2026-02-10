package main

// EmitEvent - это основной метод для генерации событий в мире.
// Он принимает GameEvent, который содержит всю необходимую информацию о событии,
// и возвращает WorldEvent, который может быть сохранён в журнале событий и отображён в UI.
// Нужен для того, чтобы обеспечить единый интерфейс для генерации событий
// , который может быть использован в любом месте симулятора мира.

func (sim *WorldSimulator) EmitEvent(event GameEvent) GameEvent {
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
func (sim *WorldSimulator) emitEventLocked(event GameEvent) GameEvent {

	sim.EventLog = append(sim.EventLog, event)
	return event
}
