// Package breaker implements the Circuit Breaker pattern. This work is
// based on github.com/rubyist/circuitbreaker, with modifications to make
// the API more Go-ish and some possible bug fixes.
//
// The Breaker will wrap a Circuit (typically one which uses remote
// services) and monitors for failures and/or time outs. When a threshold of
// failures or time outs has been reached, future calls to the function will
// not run. During this state, the breaker will periodically allow the function
// to run and, if it is successful, will start running the function again.
//
// When wrapping blocks of code with a Breaker's Call() function, a time out
// can be specified. If the time out is reached, the breaker's Fail() function
// will be called.
//
// Other types of circuit breakers can be easily built by creating a Breaker and
// adding a custom Tripper. A Tripper is called when a Breaker Fail()s and
// receives the breaker as an argument. It then returns true or false to
// indicate whether the breaker should trip.
package breaker

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/cenk/backoff"
	"github.com/lestrrat/go-circuit-breaker/breaker/internal/window"
	pdebug "github.com/lestrrat/go-pdebug"
	"github.com/pkg/errors"
)

func (s State) String() string {
	switch s {
	case Open:
		return "open"
	case Halfopen:
		return "halfopen"
	case Closed:
		return "closed"
	}
	return "(unknown:" + strconv.Itoa(int(s)) + ")"
}

// New creates a base breaker with a specified backoff, clock and TripFunc
func New(options ...Option) *breaker {
	var b breaker
	var windowTime time.Duration
	var windowBuckets int

	for _, option := range options {
		switch option.Name() {
		case "Clock":
			b.clock = option.Get().(Clock)
		case "Backoff":
			b.backoff = option.Get().(backoff.BackOff)
		case "Timeout":
			b.defaultTimeout = option.Get().(time.Duration)
		case "Tripper":
			b.tripper = option.Get().(Tripper)
		case "WindowTime":
			windowTime = option.Get().(time.Duration)
		case "WindowBuckets":
			windowBuckets = option.Get().(int)
		}
	}

	if b.tripper == nil {
		b.tripper = NilTripper
	}

	if b.clock == nil {
		b.clock = SystemClock
	}

	if b.backoff == nil {
		bo := backoff.NewExponentialBackOff()
		bo.InitialInterval = defaultInitialBackOffInterval
		bo.MaxElapsedTime = defaultBackoffMaxElapsedTime
		bo.Clock = b.clock
		bo.Reset()
		b.backoff = bo
	}

	if windowTime == 0 {
		windowTime = DefaultWindowTime
	}

	if windowBuckets == 0 {
		windowBuckets = DefaultWindowBuckets
	}

	b.nextBackOff = b.backoff.NextBackOff()
	b.counts = window.New(b.clock, windowTime, windowBuckets)
	return &b
}

func (cb *breaker) Break() {
	atomic.StoreInt32(&cb.broken, 1)
	cb.Trip()
}

func (cb *breaker) Call(circuit Circuit, options ...Option) (err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Call").BindError(&err)
		defer g.End()
	}

	timeout := cb.defaultTimeout
	for _, option := range options {
		switch option.Name() {
		case "Timeout":
			timeout = option.Get().(time.Duration)
		}
	}

	ready, st := cb.Ready()
	if !ready {
		if pdebug.Enabled {
			pdebug.Printf("Breaker not ready")
		}
		return errors.Wrap(ErrBreakerOpen, "failed to execute circuit")
	}

	switch timeout {
	case 0:
		err = circuit.Execute()
	default:
		c := make(chan error)
		d := make(chan struct{})
		defer close(d)

		go func() {
			defer close(c)
			select {
			case <-d:
				return
			case c <- circuit.Execute():
				return
			}
		}()

		select {
		case err = <-c:
		case <-cb.clock.After(timeout):
			err = errors.Wrap(ErrBreakerTimeout, "timeout reached while executing circuit")
		}
	}

	switch err {
	case nil:
		cb.success(st)
	default:
		cb.fail()
	}

	return err
}

func (cb *breaker) ConsecFailures() int64 {
	return atomic.LoadInt64(&cb.consecFailures)
}

func (cb *breaker) ErrorRate() float64 {
	return cb.counts.ErrorRate()
}

func (cb *breaker) Failures() int64 {
	return cb.counts.Failures()
}

func (cb *breaker) Ready() (isReady bool, st State) {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Ready")
		defer g.End()
	}
	st = cb.State()
	switch st {
	case Halfopen:
		if pdebug.Enabled {
			pdebug.Printf("state is halfopen")
		}
		atomic.StoreInt64(&cb.halfOpens, 0)
		fallthrough
	case Closed:
		return true, st
	}
	return false, st
}

func (cb *breaker) Reset() {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Reset")
		defer g.End()
	}

	atomic.StoreInt32(&cb.broken, 0)
	atomic.StoreInt32(&cb.tripped, 0)
	atomic.StoreInt64(&cb.halfOpens, 0)
	cb.ResetCounters()
}

func (cb *breaker) ResetCounters() {
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.Reset()
}

func (cb *breaker) State() State {
	if tripped := cb.Tripped(); !tripped {
		return Closed
	}

	if atomic.LoadInt32(&cb.broken) == 1 {
		return Open
	}

	last := atomic.LoadInt64(&cb.lastFailure)
	since := cb.clock.Now().Sub(time.Unix(last, 0))

	cb.backoffLock.Lock()
	defer cb.backoffLock.Unlock()

	if pdebug.Enabled {
		pdebug.Printf("nextBackOff %s, backoff.Stop %s, since %s", cb.nextBackOff, backoff.Stop, since)
	}
	if cb.nextBackOff != backoff.Stop && since > cb.nextBackOff {
		if pdebug.Enabled {
			pdebug.Printf("halfOpens %d", atomic.LoadInt64(&cb.halfOpens))
		}
		if atomic.CompareAndSwapInt64(&cb.halfOpens, 0, 1) {
			cb.nextBackOff = cb.backoff.NextBackOff()
			if pdebug.Enabled {
				pdebug.Printf("returning halfopen")
			}
			return Halfopen
		}
	}
	if pdebug.Enabled {
		pdebug.Printf("returning open")
	}
	return Open
}
func (cb *breaker) Successes() int64 {
	return cb.counts.Successes()
}

func (cb *breaker) Trip() {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Trip")
		defer g.End()
	}
	atomic.StoreInt32(&cb.tripped, 1)
	now := cb.clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.Unix())
}

func (cb *breaker) Tripped() bool {
	return atomic.LoadInt32(&cb.tripped) == 1
}

// fail is used to indicate a failure condition the Breaker should record.
// It will increment the failure counters and store the time of the last
// failure. If the breaker has a TripFunc it will be called, tripping the
// breaker if necessary.
func (cb *breaker) fail() {
	cb.counts.Fail()
	atomic.AddInt64(&cb.consecFailures, 1)
	now := cb.clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.Unix())
	if cb.tripper.Trip(cb) {
		cb.Trip()
	}
}

// success is used to indicate a success condition the Breaker should record.
// If the success was triggered by a retry attempt, the breaker will be Reset().
func (cb *breaker) success(st State) {
	cb.backoffLock.Lock()
	cb.backoff.Reset()
	cb.nextBackOff = cb.backoff.NextBackOff()
	cb.backoffLock.Unlock()

	if st == Halfopen {
		if pdebug.Enabled {
			pdebug.Printf("Breaker is in halfopen state, calling Reset")
		}
		cb.Reset()
	}
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.Success()
}
