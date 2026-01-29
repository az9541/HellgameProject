package main

import (
	"log"
	"sync"
)

type GameEvent struct {
	Type string
	Tick int64
	Data map[string]any
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
			log.Printf("EVENT=%s tick=%d data=%v", event.Type, event.Tick, event.Data)
		}
	}()
}
