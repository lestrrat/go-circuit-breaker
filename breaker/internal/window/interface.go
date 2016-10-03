package window

import (
	"container/ring"
	"sync"
	"time"
)

type clock interface {
	Now() time.Time
}

// Bucket holds counts of failures and successes
type Bucket struct {
	failure int64
	success int64
}

// Window maintains a ring of buckets and increments the failure and success
// counts of the current bucket. Once a specified time has elapsed, it will
// advance to the next bucket, reseting its counts. This allows the keeping of
// rolling statistics on the counts.
type Window struct {
	buckets    *ring.Ring
	bucketTime time.Duration
	bucketLock sync.RWMutex
	lastAccess time.Time
	clock      clock
}
