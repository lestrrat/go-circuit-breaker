package window

import (
	"container/ring"
	"time"
)

// Reset resets the counts to 0
func (b *Bucket) Reset() {
	b.failure = 0
	b.success = 0
}

// Fail increments the failure count
func (b *Bucket) Fail() {
	b.failure++
}

// Success increments the success count
func (b *Bucket) Success() {
	b.success++
}

// New creates a new window. windowTime is the time covering the entire
// window. windowBuckets is the number of buckets the window is divided into.
// An example: a 10 second window with 10 buckets will have 10 buckets covering
// 1 second each.
func New(c clock, windowTime time.Duration, windowBuckets int) *Window {
	buckets := ring.New(windowBuckets)
	for i := 0; i < buckets.Len(); i++ {
		buckets.Value = &Bucket{}
		buckets = buckets.Next()
	}

	bucketTime := time.Duration(windowTime.Nanoseconds() / int64(windowBuckets))
	return &Window{
		buckets:    buckets,
		bucketTime: bucketTime,
		clock:      c,
		lastAccess: c.Now(),
	}
}

// Fail records a failure in the current bucket.
func (w *Window) Fail() {
	w.bucketLock.Lock()
	b := w.getLatestBucket()
	b.Fail()
	w.bucketLock.Unlock()
}

// Success records a success in the current bucket.
func (w *Window) Success() {
	w.bucketLock.Lock()
	b := w.getLatestBucket()
	b.Success()
	w.bucketLock.Unlock()
}

// Failures returns the total number of failures recorded in all buckets.
func (w *Window) Failures() int64 {
	w.bucketLock.RLock()

	var failures int64
	w.buckets.Do(func(x interface{}) {
		b := x.(*Bucket)
		failures += b.failure
	})

	w.bucketLock.RUnlock()
	return failures
}

// Successes returns the total number of successes recorded in all buckets.
func (w *Window) Successes() int64 {
	w.bucketLock.RLock()

	var successes int64
	w.buckets.Do(func(x interface{}) {
		b := x.(*Bucket)
		successes += b.success
	})
	w.bucketLock.RUnlock()
	return successes
}

// ErrorRate returns the error rate calculated over all buckets, expressed as
// a floating point number (e.g. 0.9 for 90%)
func (w *Window) ErrorRate() float64 {
	var total int64
	var failures int64

	w.bucketLock.RLock()
	w.buckets.Do(func(x interface{}) {
		b := x.(*Bucket)
		total += b.failure + b.success
		failures += b.failure
	})
	w.bucketLock.RUnlock()

	if total == 0 {
		return 0.0
	}

	return float64(failures) / float64(total)
}

// Reset resets the count of all buckets.
func (w *Window) Reset() {
	w.bucketLock.Lock()

	w.buckets.Do(func(x interface{}) {
		x.(*Bucket).Reset()
	})
	w.bucketLock.Unlock()
}

// getLatestBucket returns the current bucket. If the bucket time has elapsed
// it will move to the next bucket, resetting its counts and updating the last
// access time before returning it. getLatestBucket assumes that the caller has
// locked the bucketLock
func (w *Window) getLatestBucket() *Bucket {
	var b *Bucket
	b = w.buckets.Value.(*Bucket)
	elapsed := w.clock.Now().Sub(w.lastAccess)

	if elapsed > w.bucketTime {
		// Reset the buckets between now and number of buckets ago. If
		// that is more that the existing buckets, reset all.
		for i := 0; i < w.buckets.Len(); i++ {
			w.buckets = w.buckets.Next()
			b = w.buckets.Value.(*Bucket)
			b.Reset()
			elapsed = time.Duration(int64(elapsed) - int64(w.bucketTime))
			if elapsed < w.bucketTime {
				// Done resetting buckets.
				break
			}
		}
		w.lastAccess = w.clock.Now()
	}
	return b
}
