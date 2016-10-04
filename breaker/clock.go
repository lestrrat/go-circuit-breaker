package breaker

import "time"

func (c systemClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

func (c systemClock) Now() time.Time {
	return time.Now()
}
