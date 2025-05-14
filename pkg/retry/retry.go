// Package retry holds a helper function for retrying a task.
package retry

import (
	"fmt"
	"log"
	"time"
)

// WithConstantWait tries to execute the task and if it fails,
// awaits the specified duration before retrying maxAttempts times.
func WithConstantWait(task string, maxAttempts int, wait time.Duration, f func() error) error {
	for attempt := 1; ; attempt++ {
		err := f()
		if err == nil {
			return nil
		}

		if attempt > maxAttempts {
			return fmt.Errorf("[%s] failed after [%d] attempts - giving up: %v", task, attempt, err)
		}

		log.Printf("[%s] attempt [%d] failed with [%v] - retrying in %s", task, attempt, err, wait)
		time.Sleep(wait)
	}
}
