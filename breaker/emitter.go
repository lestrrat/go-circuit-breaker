package breaker

import (
	"context"
	"fmt"
	"time"

	pdebug "github.com/lestrrat/go-pdebug"
)

func NewEventEmitter(cb Breaker) EventEmitter {
	return &eventEmitter{
		breaker:     cb,
		emitting:    make(chan struct{}),
		events:      make(chan BreakerEvent),
		subscribers: make(map[string]*EventSubscriber),
	}
}

func (e *eventEmitter) Events() chan BreakerEvent {
	return e.events
}

func emitEvent(e EventEmitter, ev BreakerEvent) {
	select {
	case e.Events() <- ev:
	default:
	}
}

func (e *eventEmitter) Break() {
	e.breaker.Break()
}

func (e *eventEmitter) Call(c Circuit, d time.Duration) error {
	return e.breaker.Call(c, d)
}

func (e *eventEmitter) ConsecFailures() int64 {
	return e.breaker.ConsecFailures()
}

func (e *eventEmitter) ErrorRate() float64 {
	return e.breaker.ErrorRate()
}

func (e *eventEmitter) Failures() int64 {
	return e.breaker.Failures()
}

func (e *eventEmitter) Ready() (bool, state) {
	r, st := e.breaker.Ready()
	switch st {
	case halfopen:
		defer emitEvent(e, BreakerReady)
	}
	return r, st
}

func (e *eventEmitter) Reset() {
	defer emitEvent(e, BreakerReset)
	e.breaker.Reset()
}

func (e *eventEmitter) State() state {
	return e.breaker.State()
}

func (e *eventEmitter) Successes() int64 {
	return e.breaker.Successes()
}

func (e *eventEmitter) Trip() {
	if pdebug.Enabled {
		g := pdebug.Marker("EventEmitter.Trip")
		defer g.End()
	}
	defer emitEvent(e, BreakerTripped)
	e.breaker.Trip()
}

func (e *eventEmitter) Tripped() bool {
	return e.breaker.Tripped()
}

func (e *eventEmitter) Emitting() chan struct{} {
	return e.emitting
}

// Emit does a fan-out of Breaker events
func (e *eventEmitter) Emit(ctx context.Context) {
	close(e.emitting)

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-e.events:
			if pdebug.Enabled {
				pdebug.Printf("Received event")
			}
			if !ok {
				e.events = nil
			}

			e.mutex.RLock()
			for _, l := range e.subscribers {
				select {
				case l.C <- ev:
				default:
				}
			}
			e.mutex.RUnlock()
		}
	}
}

// Subscribe starts a new subscription 
func (e *eventEmitter) Subscribe(ctx context.Context) *EventSubscriber {
	s := EventSubscriber{
		C:       make(chan BreakerEvent),
		emitter: e,
	}
	e.mutex.Lock()
	e.subscribers[fmt.Sprintf("%p", &s)] = &s
	e.mutex.Unlock()
	return &s
}

func (e *eventEmitter) remove(s *EventSubscriber) {
	e.mutex.Lock()
	delete(e.subscribers, fmt.Sprintf("%p", s))
	e.mutex.Unlock()
}

func (s *EventSubscriber) Stop() {
	s.emitter.remove(s)
}
