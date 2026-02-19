package main

// Специфический тип для событий. Позволяет избежать использования абстрактного типа string
// и обеспечивает типовую безопасность при работе с событиями.
type EventKind string

const (
	// Типы событий

	EventKindWar        EventKind = "war"
	EventKindWorld      EventKind = "world"
	EventKindTradeRoute EventKind = "trade_route"
	EventKindGeneric    EventKind = "generic"
	EventKindTick       EventKind = "tick"
)
