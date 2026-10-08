package hub

import (
	"math/rand/v2"
	"time"
)

// backoff produces exponentially growing delays with ±20% jitter, so many
// streams failing against the same NVR don't retry in lockstep.
type backoff struct {
	min, max time.Duration
	attempt  int
}

func (b *backoff) Next() time.Duration {
	d := b.min << b.attempt
	if d <= 0 || d > b.max {
		d = b.max
	} else {
		b.attempt++
	}
	jitter := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(d) * jitter)
}

func (b *backoff) Reset() {
	b.attempt = 0
}
