package sentryinbox

import (
	"math/rand/v2"
	"time"
)

const (
	// FirstRetryDelay is the delay before the second attempt.
	FirstRetryDelay = 5 * time.Second
	// MaxRetryDelay caps the delay between attempts. Retries never stop.
	MaxRetryDelay = 5 * time.Minute

	// retryJitter is the largest share of the delay that jitter adds.
	retryJitter = 0.2
)

// RetryDelay returns the delay after a failed attempt. It doubles from
// FirstRetryDelay and adds up to 20% jitter, so that entries that failed
// together do not retry together. The result never exceeds MaxRetryDelay.
// attempts counts the attempts that already ran.
func RetryDelay(attempts int, random func() float64) time.Duration {
	delay := FirstRetryDelay
	for i := 1; i < attempts && delay < MaxRetryDelay; i++ {
		delay *= 2
	}

	delay += time.Duration(random() * retryJitter * float64(delay))
	if delay > MaxRetryDelay {
		return MaxRetryDelay
	}
	return delay
}

func defaultRandom() float64 {
	return rand.Float64() // #nosec G404 -- jitter needs no cryptographic randomness.
}
