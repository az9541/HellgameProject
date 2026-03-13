package engine

import (
	"encoding/json"
	"log/slog"
)

func (gameEvent *GameEvent) UnmarshalJSON(data []byte) error {
	// Создаём алиас, повторяющий GameEvent, но с сырыми байтами для EventData
	type Alias GameEvent
	aux := &struct {
		EventData json.RawMessage
		*Alias
	}{
		Alias: (*Alias)(gameEvent),
	}

	// Декодируем всё кроме EventData
	if err := json.Unmarshal(data, &aux); err != nil {
		slog.Error("Failed to unmarshal GameEvent", "err", err)
		return err
	}

	// Проверяем, что EventData не пустой
	if len(aux.EventData) == 0 || string(aux.EventData) == "null" {
		slog.Warn("GameEvent has empty EventData", "Type", gameEvent.Type)
		return nil // Не считаем это ошибкой, просто оставляем EventData пустым
	}

	var eventData EventData

	// Определяем тип EventData на основе поля Type в GameEvent
	switch gameEvent.Type {
	case "WAR_STARTED":
		eventData = &WarStartData{}
	case "WAR_ENDED":
		eventData = &WarEndedData{}
	case "WAR_UPDATE":
		eventData = &WarUpdateData{}
	case "WAR_ABORTED":
		eventData = &WarAbortedData{}
	default:
		slog.Warn("Unknown GameEvent type is considered as GenericEvent", "Type", gameEvent.Type)
		eventData = &GenericEventData{}
	}

	// Парсим сырые байты из EventData в конкретную структуру
	if err := json.Unmarshal(aux.EventData, eventData); err != nil {
		slog.Error("Failed to unmarshal EventData for GameEvent", "Type", gameEvent.Type, "err", err)
		return err
	}

	// Применяем type assertion. Определяем конкретный тип EventData в зависимости от Type события
	switch data := eventData.(type) {
	case *WarStartData:
		gameEvent.EventData = *data
	case *WarEndedData:
		gameEvent.EventData = *data
	case *WarUpdateData:
		gameEvent.EventData = *data
	case *WarAbortedData:
		gameEvent.EventData = *data
	case *GenericEventData:
		gameEvent.EventData = *data
	default:
		slog.Warn("Parsed EventData has unknown type, leaving it as is", "Type", gameEvent.Type)
		gameEvent.EventData = data // На всякий случай сохраняем распарсенные данные, даже если тип неизвестен
	}

	return nil
}
