package engine

// EmitEvent - это основной метод для генерации событий в мире.
// Он принимает GameEvent, который содержит всю необходимую информацию о событии,
// и возвращает WorldEvent, который может быть сохранён в журнале событий и отображён в UI.
// Нужен для того, чтобы обеспечить единый интерфейс для генерации событий
// , который может быть использован в любом месте симулятора мира.

func (sim *WorldSimulator) EmitEvent(event GameEvent) GameEvent {
	sim.Mu.Lock()
	defer sim.Mu.Unlock()
	return sim.emitEventLocked(event)
}

// emitEventLocked - внутренний метод для генерации событий, который должен вызываться с уже заблокированным sim.Mu.
// Нужен для того, чтобы избежать двойного блокирования,
// если EmitEvent вызывается изнутри других методов, которые уже держат блокировку.
// emitEventLocked делает работу с EventLog и публикацией в bus.
func (sim *WorldSimulator) emitEventLocked(event GameEvent) GameEvent {
	sim.State.EventLog = append(sim.State.EventLog, event)
	if sim.EventBus != nil {
		sim.EventBus.Publish(event)
	}
	return event
}
