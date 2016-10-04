package breaker

import (
	"errors"
	"sync"
	"time"

	"github.com/cenk/backoff"
	"github.com/lestrrat/go-circuit-breaker/breaker/internal/window"
)

// Clock is an interface that defines a pluggable clock (as opposed to
// using the `time` package directly). This interface lists the only
// methods that this package cares about. You can either use your own
// implementation, or use a another library such as github.com/facebookgo/clock
type Clock interface {
	After(d time.Duration) <-chan time.Time
	Now() time.Time
}

type systemClock struct{}

// SystemClock is a simple clock using the time package
var SystemClock = systemClock{}

const (
	// DefaultWindowTime is the default time the window covers, 10 seconds.
	DefaultWindowTime time.Duration = time.Second * 10

	// DefaultWindowBuckets is the default number of buckets the window holds, 10.
	DefaultWindowBuckets = 10
)

// BreakerEvent indicates the type of event received over an event channel
type BreakerEvent int

const (
	// BreakerTripped is sent when a breaker trips
	BreakerTripped BreakerEvent = iota + 1

	// BreakerReset is sent when a breaker resets
	BreakerReset

	// BreakerFail is sent when Fail() is called
	BreakerFail

	// BreakerReady is sent when the breaker enters the half open state and is ready to retry
	BreakerReady
)

// ListenerEvent includes a reference to the circuit breaker and the event.
type ListenerEvent struct {
	CB    *Breaker
	Event BreakerEvent
}

type state int

const (
	open state = iota
	halfopen
	closed
)

var (
	defaultInitialBackOffInterval = 500 * time.Millisecond
	defaultBackoffMaxElapsedTime  = 0 * time.Second
)

// Error codes returned by Call
var (
	ErrBreakerOpen    = errors.New("breaker open")
	ErrBreakerTimeout = errors.New("breaker time out")
)

// Tripper is an interface called by a Breaker's Fail() method. It should
// determine whether the breaker should trip. By default, a Breaker has
// no Tripper
type Tripper interface {
	// Trip will receive the Breaker as an argument and returns a boolean.
	Trip(*Breaker) bool
}

// TripFunc is a type of Tripper that is represented by a function with no state
type TripFunc func(*Breaker) bool

// Breaker is the base of a circuit breaker. It maintains failure and success
// counters as well as the event subscribers.
type Breaker struct {
	backoff        backoff.BackOff
	backoffLock    sync.Mutex
	broken         int32
	clock          Clock
	consecFailures int64
	counts         *window.Window
	eventReceivers []chan BreakerEvent
	halfOpens      int64
	lastFailure    int64
	listeners      []chan ListenerEvent
	nextBackOff    time.Duration
	tripper        Tripper
	tripped        int32
}

// Circuit is the interface for those things
type Circuit interface {
	Execute() error
}

// CircuitFunc is a Cuircuit represented as a standalone function
type CircuitFunc func() error

// Option is the interface used to provide optional arguments
type Option interface {
	Name() string
	Get() interface{}
}

// Map represents a map of breakers
type Map interface {
	Get(string) (*Breaker, bool)
	Set(string, *Breaker)
}

type simpleMap struct {
	mutex sync.RWMutex
	breakers map[string]*Breaker
}
