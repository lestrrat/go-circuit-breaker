package breaker_test

import (
	"context"

	"github.com/lestrrat/go-circuit-breaker/breaker"
)

func ExampleEventEmitter() {
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
