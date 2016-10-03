package breaker

import "time"

type dumbclock struct {}

// SimpleClock returns the simplest possible object that implements
// the Clock interface
func SimpleClock() dumbclock {
	return dumbclock{}
}

func (c dumbclock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c dumbclock) Now() time.Time {
	return time.Now()
}
