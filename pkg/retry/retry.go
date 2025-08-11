// Package retry holds a helper function for retrying a task.
package retry

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type Options struct {
	Task         string
	MaxAttempts  int
	Wait         time.Duration
	InitialDelay time.Duration
	Verbose      bool
}

// WithConstantWait tries to execute the task and if it fails,
// awaits the specified duration before retrying maxAttempts times.
func WithConstantWait(f func() error, options Options) error {
	for attempt := 1; ; attempt++ {
		time.Sleep(options.InitialDelay)

		err := f()
		if err == nil {
			return nil
		}

		if attempt > options.MaxAttempts {
			return fmt.Errorf("[%s] failed after [%d] attempts - giving up: %v", options.Task, attempt, err)
		}

		if options.Verbose {
			log.Infof("[%s] attempt [%d] failed with [%v] - retrying in %s", options.Task, attempt, err, options.Wait)
		}

		time.Sleep(options.Wait)
	}
}
