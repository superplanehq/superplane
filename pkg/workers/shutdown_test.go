package workers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

func Test__DrainTasks(t *testing.T) {
	t.Run("waits for in-flight tasks to finish", func(t *testing.T) {
		sem := semaphore.NewWeighted(maxConcurrentTasks)

		var finished atomic.Int64
		const inFlight = 5

		for i := 0; i < inFlight; i++ {
			//
			// Mirrors a worker: take a slot, then do the work in a goroutine
			// that releases the slot when it is done.
			//
			_ = sem.Acquire(context.Background(), 1)

			go func() {
				defer sem.Release(1)

				time.Sleep(100 * time.Millisecond)
				finished.Add(1)
			}()
		}

		assert.Less(t, finished.Load(), int64(inFlight), "tasks should still be running")

		drainTasks(sem, maxConcurrentTasks)

		assert.Equal(t, int64(inFlight), finished.Load(), "drain must not return while work is in flight")
	})

	t.Run("returns immediately when nothing is in flight", func(t *testing.T) {
		sem := semaphore.NewWeighted(maxConcurrentCleanupTasks)

		start := time.Now()
		drainTasks(sem, maxConcurrentCleanupTasks)

		assert.Less(t, time.Since(start), time.Second)
	})
}
