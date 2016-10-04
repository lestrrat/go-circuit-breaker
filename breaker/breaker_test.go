package breaker_test

import (
	"testing"
	"time"

	"github.com/cenk/backoff"
	"github.com/lestrrat/go-circuit-breaker/breaker"
	"github.com/stretchr/testify/assert"
)

func defaultBackOff(c breaker.Clock) backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Millisecond
	bo.Clock = c
	bo.Reset()
	return bo
}

func newBreaker(options ...breaker.Option) *breaker.Breaker {
	var c breaker.Clock
	var bo backoff.BackOff
	for _, option := range options {
		switch option.Name() {
		case "Clock":
			c = option.Get().(breaker.Clock)
		case "Backoff":
			bo = option.Get().(backoff.BackOff)
		}
	}

	if c == nil {
		c = breaker.SystemClock
		options = append(options, breaker.WithClock(c))
	}

	if bo == nil {
		bo = defaultBackOff(c)
		options = append(options, breaker.WithBackOff(bo))
	}

	return breaker.New(options...)
}

func TestBreakerTripping(t *testing.T) {
	cb := newBreaker()
	if !assert.False(t, cb.Tripped(), "expected breaker to not be tripped") {
		return
	}

	cb.Trip()

	if !assert.True(t, cb.Tripped(), "expected breaker to be tripped") {
		return
	}

	cb.Reset()
	if !assert.False(t, cb.Tripped(), "expected breaker to have been reset") {
		return
	}
}

func TestErrorRate(t *testing.T) {
	cb := newBreaker()
	if er := cb.ErrorRate(); er != 0.0 {
		t.Fatalf("expected breaker with no samples to have 0 error rate, got %f", er)
	}
}
