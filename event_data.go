package main

// === СТРУКТУРЫ ДЛЯ СОБЫТИЙ МИРА ===

// Ещё одна структура для прототипирования событий, для которых лень делать отдельные структуры данных.
type GenericEventData struct {
	EventKind EventKind
	EventData map[string]any
}

type WarStartData struct {
	Attacker              string
	Defender              string
	Domain                string
	DomainID              string
	Reason                string
	ActualDefenderForce   float64
	EstimateDefenderForce float64
	AttackerCommitted     float64
	DefenderCommitted     float64
	Ratio                 float64
	MinStrengthRatio      float64
}

type WarUpdateData struct {
	Attacker          string
	Defender          string
	Domain            string
	Momentum          float64
	AttackerMorale    float64
	DefenderMorale    float64
	AttackerForce     float64
	DefenderForce     float64
	AttackerLossesPct float64
	DefenderLossesPct float64
	ForceRatio        float64
}

type WarEndedData struct {
	Attacker            string
	Defender            string
	Domain              string
	Reason              string
	WinnerID            string
	LoserID             string
	AttackerLossesPct   float64
	DefenderLossesPct   float64
	AttackerMorale      float64
	DefenderMorale      float64
	AttackerForceRemain float64
	DefenderForceRemain float64
}

type WarAbortedData struct {
	Attacker              string
	Defender              string
	Domain                string
	DomainID              string
	Reason                string
	ActualDefenderForce   float64
	EstimateDefenderForce float64
	AttackerCommitted     float64
	DefenderCommitted     float64
	Ratio                 float64
	MinStrengthRatio      float64
}

// === ИНТЕРФЕЙСЫ ДЛЯ СОБЫТИЙ МИРА ===

// У нас может быть множество типов событий. Поэтому любой тип, который хочет считаться событием,
// должен реализовать интерфейс EventData. Это позволяет нам обрабатывать разные типы событий единообразно.
// Сам по себе интерфейс минималистичный - он содержит один метод Kind(),
// который возвращает EventKind(алиас), идентифицирующую тип события.
// Data - это внутренняя информация о событии, которая может быть полезна для логики игры, но не обязательно должна отображаться пользователю.
type EventData interface {
	Kind() EventKind
}

// === ИМПЛЕМЕНТАЦИЯ ИНТЕРФЕЙСОВ
// Имплементация внутренних данных для Generic-эвентов
func (d GenericEventData) Kind() EventKind {
	if d.EventKind != "" {
		return d.EventKind
	}
	return EventKindGeneric
}

// Имплементация интерфейсов для WarEventData
// Внутренние данные для эвентов войны
func (d WarStartData) Kind() EventKind {
	return EventKindWar
}

func (d WarUpdateData) Kind() EventKind {
	return EventKindWar
}

func (d WarEndedData) Kind() EventKind {
	return EventKindWar
}

func (d WarAbortedData) Kind() EventKind {
	return EventKindWar
}
