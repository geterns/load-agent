package dummy

import (
	"time"
)

type DummyWriter struct {
	FirstByteArrived     *bool
	FirstByteArrivalTime *time.Time
	LastDataArrivalTime  *time.Time
	MaxWaitDuration      *time.Duration
}

func (c DummyWriter) Write(p []byte) (n int, err error) {
	// Measure first byte arrival time
	if !*c.FirstByteArrived {
		*c.FirstByteArrived = true
		*c.FirstByteArrivalTime = time.Now()
		*c.LastDataArrivalTime = time.Now()
	} else {
		waitDuration := time.Now().Sub(*c.LastDataArrivalTime)
		*c.LastDataArrivalTime = time.Now()

		if waitDuration > *c.MaxWaitDuration {
			*c.MaxWaitDuration = waitDuration
		}
	}

	return len(p), nil
}
