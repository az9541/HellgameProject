package engine

import (
	"log/slog"
	"sync"
)

type IBuilderGameEvent interface {
	SetType(string) IBuilderGameEvent
	SetTick(int64) IBuilderGameEvent
	SetData(EventData) IBuilderGameEvent
	SetKind(EventKind) IBuilderGameEvent
	Build() GameEvent
}

type BuilderWarEvent struct {
	Type      string
	Tick      int64
	EventData EventData
	EventKind EventKind
}

func (b *BuilderWarEvent) SetKind(k EventKind) IBuilderGameEvent {
	b.EventKind = k
	return b
}

func (b *BuilderWarEvent) SetType(t string) IBuilderGameEvent {
	b.Type = t
	return b
}

func (b *BuilderWarEvent) SetTick(t int64) IBuilderGameEvent {
	b.Tick = t
	return b
}

func (b *BuilderWarEvent) SetData(d EventData) IBuilderGameEvent {
	b.EventData = d
	return b
}

func (b *BuilderWarEvent) Build() GameEvent {
	return GameEvent{
		Type:      b.Type,
		Tick:      b.Tick,
		EventData: b.EventData,
		EventKind: b.EventKind,
	}
}

func NewBuilderWarEvent() *BuilderWarEvent {
	return &BuilderWarEvent{EventKind: EventKindWar}
}

type BuilderGenericEvent struct {
	Type      string
	Tick      int64
	EventData EventData
	EventKind EventKind
}

func NewBuilderGenericEvent() *BuilderGenericEvent {
	return &BuilderGenericEvent{EventKind: EventKindGeneric}
}

func (b *BuilderGenericEvent) SetKind(k EventKind) IBuilderGameEvent {
	b.EventKind = k
	return b
}

func (b *BuilderGenericEvent) SetType(t string) IBuilderGameEvent {
	b.Type = t
	return b
}

func (b *BuilderGenericEvent) SetTick(t int64) IBuilderGameEvent {
	b.Tick = t
	return b
}

func (b *BuilderGenericEvent) SetData(d EventData) IBuilderGameEvent {
	b.EventData = d
	return b
}

func (b *BuilderGenericEvent) Build() GameEvent {
	return GameEvent{
		Type:      b.Type,
		Tick:      b.Tick,
		EventData: b.EventData,
		EventKind: b.EventKind,
	}
}

type GameEvent struct {
	Type      string
	Tick      int64
	EventData EventData
	EventKind EventKind
}

type EventPublisher struct {
	mu          sync.RWMutex
	subscribers []chan GameEvent
}

func NewEventPublisher() *EventPublisher {
	return &EventPublisher{}
}

func (ep *EventPublisher) Subscribe(buffer int) chan GameEvent {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ch := make(chan GameEvent, buffer)
	ep.subscribers = append(ep.subscribers, ch)
	return ch
}

func (ep *EventPublisher) Publish(event GameEvent) {
	if ep == nil {
		return
	}
	ep.mu.RLock()
	subs := append([]chan GameEvent(nil), ep.subscribers...)
	ep.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func (ep *EventPublisher) Unsubscribe(ch chan GameEvent) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	for i, s := range ep.subscribers {
		if s == ch {
			ep.subscribers = append(ep.subscribers[:i], ep.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

func StartEventLogger(bus *EventPublisher, buffer int) {
	if bus == nil {
		return
	}
	ch := bus.Subscribe(buffer)
	go func() {
		for event := range ch {
			slog.Info("EVENT",
				"event_type", event.Type,
				"tick", event.Tick,
				"data", event.EventData,
			)
		}
	}()
}
