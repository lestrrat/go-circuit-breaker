package breaker

// Trip return true if the TripFunc thinks the failure
// state has reached the point where the circuit
// breaker should be tripped
func (f TripFunc) Trip(cb Breaker) bool {
	return f(cb)
}

// NilTripper is a Tripper that always returns false
var NilTripper = TripFunc(func(cb Breaker) bool {
	return false
})

// ThresholdTripper returns a Tripper that trips whenever
// the failure count meets the given threshold.
func ThresholdTripper(threshold int64) Tripper {
	return TripFunc(func(cb Breaker) bool {
		return cb.Failures() >= threshold
	})
}

// ConsecutiveTripper returns a Tripper that trips whenever
// the *consecutive* failure count meets the given threshold.
func ConsecutiveTripper(threshold int64) Tripper {
	return TripFunc(func(cb Breaker) bool {
		return cb.ConsecFailures() >= threshold
	})
}

// RateTripper returns a Tripper that trips whenever the
// error rate hits the given threshold.
//
// The error rate is calculated as such:
// f = number of failures
// s = number of successes
// e = f / (f + s)
//
// The error rate is calculated over a sliding window of 10 seconds (by default)
// This Tripper will not trip until there has been at least minSamples events.
func RateTripper(rate float64, minSamples int64) Tripper {
	return TripFunc(func(cb Breaker) bool {
		samples := cb.Failures() + cb.Successes()
		return samples >= minSamples && cb.ErrorRate() >= rate
	})
}
