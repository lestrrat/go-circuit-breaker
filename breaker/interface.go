package breaker

import (
	"context"
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

// Event indicates the type of event received over an event channel
type Event int

const (
	// TrippedEvent is sent when a breaker trips
	TrippedEvent Event = iota + 1

	// ResetEvent is sent when a breaker resets
	ResetEvent

	// FailEvent is sent when Fail() is called
	FailEvent

	// ReadyEvent is sent when the breaker enters the half open state and is ready to retry
	ReadyEvent
)

// State describes the current state of the Breaker
type State int

// The various states that the Breaker can take
const (
	Open State = iota
	Halfopen
	Closed
)

var (
	defaultInitialBackOffInterval = 500 * time.Millisecond
	defaultBackoffMaxElapsedTime  = 0 * time.Second
)

// Error codes returned by Call
var (
	ErrBreakerOpen    = breakerOpenErr{}
	ErrBreakerTimeout = breakerTimeoutErr{}
)

// Tripper is an interface called by a Breaker's Fail() method. It should
// determine whether the breaker should trip. By default, a Breaker has
// no Tripper
type Tripper interface {
	// Trip will receive the Breaker as an argument and returns a boolean.
	Trip(Breaker) bool
}

// TripFunc is a type of Tripper that is represented by a function with no state
type TripFunc func(Breaker) bool

// Breaker describes the interface of a circuit breaker. It maintains
// failure and success counters and state information
type Breaker interface {
	// Break trips the circuit breaker and prevents it from auto resetting.
	// Use this when manual control over the circuit breaker state is needed.
	Break()

	// Call wraps a function the Breaker will protect. A failure is recorded
	// whenever the function returns an error.
	//
	// `WithTimeout` may be specified in the options to override the default
	// timeout associated with the breaker. If the called function takes longer
	// than timeout to run, a failure will be recorded.
	Call(Circuit, ...Option) error

	// ConsecFailures returns the number of consecutive failures that
	// have occured.
	ConsecFailures() int64

	// ErrorRate returns the current error rate of the Breaker, expressed
	// as a floating point number (e.g. 0.9 for 90%), since the last time
	// the breaker was Reset.
	ErrorRate() float64

	// Failures returns the number of failures for this circuit breaker.
	Failures() int64

	// Ready will return true if the circuit breaker is ready to call the
	// function. It will be ready if the breaker is in a reset state, or if
	// it is time to retry the call for auto resetting.
	//
	// Note that the method has side effects. If you are only interested in
	// querying for the current state without causing side effects,
	// you should use State()
	Ready() (bool, State)

	// Reset will reset the circuit breaker. After Reset() is called,
	// Tripped() will return false.
	Reset()

	// ResetCounters will reset only the failures, consecFailures,
	// and success counters
	ResetCounters()

	// State returns the state of the Breaker. The states available are:
	// Closed - the circuit is in a reset state and is operational
	// Open - the circuit is in a tripped state
	// Halfopen - the circuit is in a tripped state but the reset timeout has passed
	State() State

	// Successes returns the number of successes for this circuit breaker.
	Successes() int64

	// Trip will trip the circuit breaker. After Trip() is called, Tripped()
	// willreturn true.
	Trip()

	// Tripped returns true if the circuit breaker is tripped, false
	// if it is reset.
	Tripped() bool
}

// EventSubscription describes a subscription to an EventEmitter
type EventSubscription struct {
	C       chan Event
	emitter *eventEmitter
}

// EventEmitter is used to wrap a Breaker object so that useful
// notifications can be received from it.
type EventEmitter interface {
	Breaker
	Emitting() chan struct{}
	Emit(context.Context)
	Events() chan Event
	Subscribe(context.Context) *EventSubscription
}

type eventEmitter struct {
	breaker     Breaker
	emitting    chan struct{}
	events      chan Event
	mutex       sync.RWMutex
	subscribers map[string]*EventSubscription
}

type breaker struct {
	backoff        backoff.BackOff
	backoffLock    sync.Mutex
	broken         int32
	clock          Clock
	consecFailures int64
	counts         *window.Window
	defaultTimeout time.Duration
	halfOpens      int64
	lastFailure    int64
	nextBackOff    time.Duration
	tripper        Tripper
	tripped        int32
}

// Circuit is the interface for things that can be Call'ed
// and protected by the Breaker
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
	Get(string) (Breaker, bool)
	Set(string, Breaker)
}

type simpleMap struct {
	mutex    sync.RWMutex
	breakers map[string]Breaker
}
