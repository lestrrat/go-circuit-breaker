package breaker

import (
	"github.com/cenk/backoff"
	"github.com/lestrrat/go-circuit-breaker/internal/option"
)

// WithClock is used specify the clock used by the circuir breaker.
// Normally, this is only used for testing
func WithClock(v Clock) Option {
	return option.NewValue("Clock", v)
}

// WithBackOff is used to specify the backoff policy that is used when
// determining if the breaker should attempt to retry. `Breaker` objects
// will use an exponential backoff policy by default.
func WithBackOff(v backoff.BackOff) Option {
	return option.NewValue("Backoff", v)
}

// WithTripper is used to specify the tripper that is used when
// determining when the breaker should trip.
func WithTripper(v Tripper) Option {
	return option.NewValue("Tripper", v)
}
