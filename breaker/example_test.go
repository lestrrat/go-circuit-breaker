package breaker_test

import (
	"context"
	"time"

	"github.com/cenk/backoff"
	"github.com/lestrrat/go-circuit-breaker/breaker"
)

func Example() {
	// Need to initialize a clock to use for the
	// backoff AND the breaker
	c := breaker.SystemClock

	// Create a custom backoff strategy
	bo := backoff.NewExponentialBackOff()
	bo.Clock = c

	cb := breaker.New(
		breaker.WithBackOff(bo),
		breaker.WithClock(c),
		breaker.WithTimeout(10*time.Second),
		breaker.WithTripper(breaker.ThresholdTripper(10)),
	)

	err := cb.Call(breaker.CircuitFunc(func() error {
		// call that may trip the breaker
		return nil
	}))
	// err will be non-nill if either the circuit returns
	// an error, or the breaker is in an Open state
	_ = err
}

func ExampleEventEmitter() {
	// Use emitter to receive notifications of events
	// such as TrippedEvent, ReadyEvent, ResetEvent, etc.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cb := breaker.NewEventEmitter(breaker.New())
	s := cb.Subscribe(ctx)

	for {
		select {
		case e := <-s.C:
			// received event
			_ = e
		}
	}
}
