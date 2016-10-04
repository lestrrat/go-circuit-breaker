// Package breaker implements the Circuit Breaker pattern. This work is
// based on github.com/rubyist/circuitbreaker, with modifications to make
// the API more Go-ish and some possible bug fixes.
//
// The Breaker will wrap a function call (typically one which uses remote
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

// Trip will trip the circuit breaker. After Trip() is called, Tripped() will
// return true.
func (cb *breaker) Trip() {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Trip")
		defer g.End()
	}
	atomic.StoreInt32(&cb.tripped, 1)
	now := cb.clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.Unix())
}

// Reset will reset the circuit breaker. After Reset() is called, Tripped() will
// return false.
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

// ResetCounters will reset only the failures, consecFailures, and success counters
func (cb *breaker) ResetCounters() {
	atomic.StoreInt64(&cb.consecFailures, 0)
	cb.counts.Reset()
}

// Tripped returns true if the circuit breaker is tripped, false if it is reset.
func (cb *breaker) Tripped() bool {
	return atomic.LoadInt32(&cb.tripped) == 1
}

// Break trips the circuit breaker and prevents it from auto resetting. Use this when
// manual control over the circuit breaker state is needed.
func (cb *breaker) Break() {
	atomic.StoreInt32(&cb.broken, 1)
	cb.Trip()
}

// Failures returns the number of failures for this circuit breaker.
func (cb *breaker) Failures() int64 {
	return cb.counts.Failures()
}

// ConsecFailures returns the number of consecutive failures that have occured.
func (cb *breaker) ConsecFailures() int64 {
	return atomic.LoadInt64(&cb.consecFailures)
}

// Successes returns the number of successes for this circuit breaker.
func (cb *breaker) Successes() int64 {
	return cb.counts.Successes()
}

// fail is used to indicate a failure condition the Breaker should record. It will
// increment the failure counters and store the time of the last failure. If the
// breaker has a TripFunc it will be called, tripping the breaker if necessary.
func (cb *breaker) fail() {
	cb.counts.Fail()
	atomic.AddInt64(&cb.consecFailures, 1)
	now := cb.clock.Now()
	atomic.StoreInt64(&cb.lastFailure, now.Unix())
	if cb.tripper.Trip(cb) {
		cb.Trip()
	}
}

// success is used to indicate a success condition the Breaker should record. If
// the success was triggered by a retry attempt, the breaker will be Reset().
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

// ErrorRate returns the current error rate of the Breaker, expressed as a floating
// point number (e.g. 0.9 for 90%), since the last time the breaker was Reset.
func (cb *breaker) ErrorRate() float64 {
	return cb.counts.ErrorRate()
}

// Ready will return true if the circuit breaker is ready to call the function.
//
// It will be ready if the breaker is in a reset state, or if it is time to retry
// the call for auto resetting. Note that this means that the method has
// side effects. If you are only interested in querying for the current state,
// you should use State()
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

// Call wraps a function the Breaker will protect. A failure is recorded
// whenever the function returns an error. If the called function takes longer
// than timeout to run, a failure will be recorded.
func (cb *breaker) Call(circuit Circuit, timeout time.Duration) (err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("Breaker.Call").BindError(&err)
		defer g.End()
	}

	ready, st := cb.Ready()
	if !ready {
		if pdebug.Enabled {
			pdebug.Printf("Breaker not ready")
		}
		return ErrBreakerOpen
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
			err = ErrBreakerTimeout
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

// State returns the state of the TrippableBreaker. The states available are:
// closed - the circuit is in a reset state and is operational
// open - the circuit is in a tripped state
// halfopen - the circuit is in a tripped state but the reset timeout has passed
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
