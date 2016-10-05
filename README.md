# go-circuit-breaker

Circuit Breaker Pattern for Go

[![Build Status](https://travis-ci.org/lestrrat/go-circuit-breaker.svg?branch=master)](https://travis-ci.org/lestrrat/go-circuit-breaker)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-circuit-breaker?status.svg)](https://godoc.org/github.com/lestrrat/go-circuit-breaker)

# SYNOPSIS

```go
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
```

# DESCRIPTION

This is a fork of https://github.com/rubyist/circuitbreaker.

There were few issues (timing sensitive tests) and tweaks that I wanted to see
(e.g. separating the event subscription from the core breaker, and ways to
pass optional parameters), but they mostly
required API changes, which is a very hard to thing to press for.

So there it is. Yet another circuit breaker :) .

# CONTRIBUTING

PRs are welcome. If you have a new patch, please attach a test.
If you are suggesting new API, or change, please attach a failing test
to demonstrate what you are looking for. In short: please attach a test.
