package breaker

import (
	"context"
	"fmt"

	pdebug "github.com/lestrrat/go-pdebug"
)

// NewEventEmitter wraps Breaker and creates an EventEmitter
// (which also satisfies the Breaker interface) that can
// generate events.
func NewEventEmitter(cb Breaker) EventEmitter {
	return &eventEmitter{
		breaker:     cb,
		emitting:    make(chan struct{}),
		events:      make(chan Event),
		subscribers: make(map[string]*EventSubscription),
	}
}

func (e *eventEmitter) Events() chan Event {
	return e.events
}

func emitEvent(e EventEmitter, ev Event) {
	select {
	case e.Events() <- ev:
	default:
	}
}

func (e *eventEmitter) Break() {
	e.breaker.Break()
}

func (e *eventEmitter) Call(c Circuit, options ...Option) error {
	return e.breaker.Call(c, options...)
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

func (e *eventEmitter) Ready() (bool, State) {
	r, st := e.breaker.Ready()
	switch st {
	case Halfopen:
		defer emitEvent(e, ReadyEvent)
	}
	return r, st
}

func (e *eventEmitter) Reset() {
	defer emitEvent(e, ResetEvent)
	e.breaker.Reset()
}

func (e *eventEmitter) ResetCounters() {
	e.breaker.ResetCounters()
}

func (e *eventEmitter) State() State {
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
	defer emitEvent(e, TrippedEvent)
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
func (e *eventEmitter) Subscribe(ctx context.Context) *EventSubscription {
	s := EventSubscription{
		C:       make(chan Event),
		emitter: e,
	}
	e.mutex.Lock()
	e.subscribers[fmt.Sprintf("%p", &s)] = &s
	e.mutex.Unlock()
	return &s
}

func (e *eventEmitter) remove(s *EventSubscription) {
	e.mutex.Lock()
	delete(e.subscribers, fmt.Sprintf("%p", s))
	e.mutex.Unlock()
}

// Stop removes the subscription from the associated EventEmitter
// and stops receiving events
func (s *EventSubscription) Stop() {
	s.emitter.remove(s)
}
