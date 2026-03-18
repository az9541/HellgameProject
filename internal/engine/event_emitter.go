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

// MaxEventLogSize определяет максимальное количество последних событий, хранимых в State
// Это нужно, чтобы стейт (и сохранения в БД) не разрастались до гигабайт.
const MaxEventLogSize = 500

// emitEventLocked - внутренний метод для генерации событий, который должен вызываться с уже заблокированным sim.Mu.
// Нужен для того, чтобы избежать двойного блокирования,
// если EmitEvent вызывается изнутри других методов, которые уже держат блокировку.
// emitEventLocked делает работу с EventLog и публикацией в bus.
func (sim *WorldSimulator) emitEventLocked(event GameEvent) GameEvent {
	sim.State.EventLog = append(sim.State.EventLog, event)

	// Ограничиваем размер (Ring Buffer) для MVP
	if len(sim.State.EventLog) > MaxEventLogSize {
		// Сдвигаем окно (Go сам потом перевыделит память, когда упрётся в capacity)
		sim.State.EventLog = sim.State.EventLog[len(sim.State.EventLog)-MaxEventLogSize:]
	}

	if sim.EventBus != nil {
		sim.EventBus.Publish(event)
	}
	return event
}
