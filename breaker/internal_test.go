package breaker

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenk/backoff"
	"github.com/facebookgo/clock"
	"github.com/stretchr/testify/assert"
)

func defaultBackOff(c Clock) backoff.BackOff {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = time.Millisecond
	bo.Clock = c
	bo.Reset()
	return bo
}

func TestSanity(t *testing.T) {
	var b Breaker
	b = &breaker{}
	_ = b
}

func newBreaker(options ...Option) Breaker {
	var c Clock
	var bo backoff.BackOff
	for _, option := range options {
		switch option.Name() {
		case "Clock":
			c = option.Get().(Clock)
		case "Backoff":
			bo = option.Get().(backoff.BackOff)
		}
	}

	if c == nil {
		c = SystemClock
		options = append(options, WithClock(c))
	}

	if bo == nil {
		bo = defaultBackOff(c)
		options = append(options, WithBackOff(bo))
	}

	return New(options...)
}

func TestTrippableBreakerState(t *testing.T) {
	c := clock.NewMock()
	bo := defaultBackOff(c)
	cb := newBreaker(
		WithBackOff(bo),
		WithClock(c),
	)

	if r, _ := cb.Ready(); !assert.True(t, r, "expected breaker to be ready") {
		return
	}

	cb.Trip()
	if r, _ := cb.Ready(); !assert.False(t, r, "expected breaker to not be ready") {
		return
	}

	c.Add(bo.NextBackOff() + 1)
	if r, _ := cb.Ready(); !assert.True(t, r, "expected breaker to be ready after reset timeout") {
		return
	}

	cb.(*breaker).fail()
	c.Add(bo.NextBackOff() + 1)
	if r, _ := cb.Ready(); !assert.True(t, r, "expected breaker to be ready after reset timeout, post failure") {
		return
	}
}

func TestTrippableBreakerManualBreak(t *testing.T) {
	c := clock.NewMock()
	bo := defaultBackOff(c)
	cb := newBreaker(
		WithBackOff(bo),
		WithClock(c),
	)
	cb.Break()
	c.Add(bo.NextBackOff() + 1)

	if r, _ := cb.Ready(); !assert.False(t, r, "expected breaker to still be tripped") {
		return
	}

	cb.Reset()
	cb.Trip()
	c.Add(bo.NextBackOff() + 1)
	if r, _ := cb.Ready(); !assert.True(t, r, "expected breaker to be ready") {
		return
	}
}

func TestThresholdBreaker(t *testing.T) {
	cb := newBreaker(WithTripper(ThresholdTripper(2)))

	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	cb.(*breaker).fail()
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to still be open")
	}

	cb.(*breaker).fail()
	if !cb.Tripped() {
		t.Fatal("expected threshold breaker to be tripped")
	}

	cb.Reset()
	if failures := cb.Failures(); failures != 0 {
		t.Fatalf("expected reset to set failures to 0, got %d", failures)
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}
}

func TestConsecutiveBreaker(t *testing.T) {
	cb := newBreaker(WithTripper(ConsecutiveTripper(3)))

	if cb.Tripped() {
		t.Fatal("expected consecutive breaker to be open")
	}

	cb.(*breaker).fail()
	cb.(*breaker).success(cb.(*breaker).State())
	cb.(*breaker).fail()
	cb.(*breaker).fail()
	if cb.Tripped() {
		t.Fatal("expected consecutive breaker to be open")
	}
	cb.(*breaker).fail()
	if !cb.Tripped() {
		t.Fatal("expected consecutive breaker to be tripped")
	}
}

func TestThresholdBreakerCalling(t *testing.T) {
	circuit := CircuitFunc(func() error {
		return errors.New("error")
	})

	cb := newBreaker(WithTripper(ThresholdTripper(2)))

	err := cb.Call(circuit, 0) // First failure
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if cb.Tripped() {
		t.Fatal("expected threshold breaker to be open")
	}

	err = cb.Call(circuit, 0) // Second failure trips
	if err == nil {
		t.Fatal("expected threshold breaker to error")
	}
	if !cb.Tripped() {
		t.Fatal("expected threshold breaker to be tripped")
	}
}

func TestThresholdBreakerResets(t *testing.T) {
	called := 0
	success := false
	circuit := CircuitFunc(func() error {
		t.Logf("circuit called %d", called)
		if called == 0 {
			called++
			return errors.New("error")
		}
		success = true
		t.Logf("circuit success")
		return nil
	})

	c := clock.NewMock()
	bo := defaultBackOff(c)
	cb := newBreaker(
		WithClock(c),
		WithTripper(ThresholdTripper(1)),
	)

	t.Logf("First call to circuit, should fail")
	if !assert.Error(t, cb.Call(circuit, 0), "Expected cb to return an error") {
		return
	}

	c.Add(bo.NextBackOff() + time.Second)
	for i := 0; i < 4; i++ {
		t.Logf("Attempting subsequent call %d, should succeed", i)
		if !assert.NoError(t, cb.Call(circuit, 0), "Expected cb to be successful (#%d)", i) {
			return
		}

		if !assert.True(t, success, "Expected cb to have been reset") {
			return
		}
	}
}

func TestTimeoutBreaker(t *testing.T) {
	wait := make(chan struct{})

	c := clock.NewMock()
	called := int32(0)

	circuit := CircuitFunc(func() error {
		wait <- struct{}{}
		atomic.AddInt32(&called, 1)
		<-wait
		return nil
	})

	cb := newBreaker(
		WithClock(c),
		WithTripper(ThresholdTripper(1)),
	)

	errc := make(chan error)
	go func() { errc <- cb.Call(circuit, time.Millisecond) }()

	<-wait
	c.Add(time.Millisecond * 3)
	wait <- struct{}{}

	err := <-errc
	if err == nil {
		t.Fatal("expected timeout breaker to return an error")
	}

	go cb.Call(circuit, time.Millisecond)
	<-wait
	c.Add(time.Millisecond * 3)
	wait <- struct{}{}

	if !cb.Tripped() {
		t.Fatal("expected timeout breaker to be open")
	}
}

func TestRateBreakerTripping(t *testing.T) {
	cb := newBreaker(WithTripper(RateTripper(0.5, 4)))
	cb.(*breaker).success(cb.(*breaker).State())
	cb.(*breaker).success(cb.(*breaker).State())
	cb.(*breaker).fail()
	cb.(*breaker).fail()

	if !cb.Tripped() {
		t.Fatal("expected rate breaker to be tripped")
	}

	if er := cb.ErrorRate(); er != 0.5 {
		t.Fatalf("expected error rate to be 0.5, got %f", er)
	}
}

func TestRateBreakerSampleSize(t *testing.T) {
	cb := newBreaker(WithTripper(RateTripper(0.5, 100)))
	cb.(*breaker).fail()

	if cb.Tripped() {
		t.Fatal("expected rate breaker to not be tripped yet")
	}
}

func TestRateBreakerResets(t *testing.T) {
	serviceError := errors.New("service error")

	called := 0
	success := false
	circuit := CircuitFunc(func() error {
		if called < 4 {
			called++
			return serviceError
		}
		success = true
		return nil
	})

	c := clock.NewMock()
	bo := defaultBackOff(c)
	cb := newBreaker(
		WithBackOff(bo),
		WithClock(c),
		WithTripper(RateTripper(0.5, 4)),
	)
	var err error
	for i := 0; i < 4; i++ {
		err = cb.Call(circuit, 0)
		if err == nil {
			t.Fatal("Expected cb to return an error (closed breaker, service failure)")
		} else if err != serviceError {
			t.Fatal("Expected cb to return error from service (closed breaker, service failure)")
		}
	}

	err = cb.Call(circuit, 0)
	if err == nil {
		t.Fatal("Expected cb to return an error (open breaker)")
	} else if err != ErrBreakerOpen {
		t.Fatal("Expected cb to return open open breaker error (open breaker)")
	}

	c.Add(bo.NextBackOff() + time.Second)
	err = cb.Call(circuit, 0)
	if err != nil {
		t.Fatal("Expected cb to be successful")
	}

	if !success {
		t.Fatal("Expected cb to have been reset")
	}
}

func TestNeverRetryAfterBackoffStops(t *testing.T) {
	cb := newBreaker(WithBackOff(&backoff.StopBackOff{}))
	cb.Trip()

	// circuit should be open and never retry again
	// when nextBackoff is backoff.Stop
	called := 0
	cb.Call(CircuitFunc(func() error {
		called = 1
		return nil
	}), 0)

	if called == 1 {
		t.Fatal("Expected cb to never retry")
	}
}

func TestBreakerCounts(t *testing.T) {
	cb := newBreaker()

	cb.(*breaker).fail()
	if failures := cb.Failures(); failures != 1 {
		t.Fatalf("expected failure count to be 1, got %d", failures)
	}

	cb.(*breaker).fail()
	if consecFailures := cb.ConsecFailures(); consecFailures != 2 {
		t.Fatalf("expected 2 consecutive failures, got %d", consecFailures)
	}

	cb.(*breaker).success(cb.(*breaker).State())
	if successes := cb.Successes(); successes != 1 {
		t.Fatalf("expected success count to be 1, got %d", successes)
	}
	if consecFailures := cb.ConsecFailures(); consecFailures != 0 {
		t.Fatalf("expected 0 consecutive failures, got %d", consecFailures)
	}

	cb.Reset()
	if failures := cb.Failures(); failures != 0 {
		t.Fatalf("expected failure count to be 0, got %d", failures)
	}
	if successes := cb.Successes(); successes != 0 {
		t.Fatalf("expected success count to be 0, got %d", successes)
	}
	if consecFailures := cb.ConsecFailures(); consecFailures != 0 {
		t.Fatalf("expected 0 consecutive failures, got %d", consecFailures)
	}
}
