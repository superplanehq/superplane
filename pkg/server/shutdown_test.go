package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ShutdownTimeout(t *testing.T) {
	t.Run("defaults when unset", func(t *testing.T) {
		t.Setenv("SHUTDOWN_TIMEOUT", "")
		assert.Equal(t, defaultShutdownTimeout, shutdownTimeout())
	})

	t.Run("reads the environment", func(t *testing.T) {
		t.Setenv("SHUTDOWN_TIMEOUT", "5s")
		assert.Equal(t, 5*time.Second, shutdownTimeout())
	})

	t.Run("falls back when the value does not parse", func(t *testing.T) {
		t.Setenv("SHUTDOWN_TIMEOUT", "banana")
		assert.Equal(t, defaultShutdownTimeout, shutdownTimeout())
	})

	t.Run("stays under the Kubernetes default grace period", func(t *testing.T) {
		//
		// The chart does not set terminationGracePeriodSeconds, so Kubernetes
		// sends SIGKILL after 30s. The drain has to finish before that.
		//
		assert.Less(t, defaultShutdownTimeout, 30*time.Second)
	})
}

func Test__StartWorker(t *testing.T) {
	t.Run("cancels the worker and waits for it", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup

		stopped := make(chan struct{})
		startWorker(ctx, &wg, func(workerCtx context.Context) {
			<-workerCtx.Done()

			//
			// Stand in for work that outlives the signal, so the assertion below
			// fails if Start stops waiting for the worker.
			//
			time.Sleep(50 * time.Millisecond)
			close(stopped)
		})

		select {
		case <-stopped:
			require.Fail(t, "worker stopped before the context was cancelled")
		case <-time.After(20 * time.Millisecond):
		}

		cancel()
		waitForShutdown(&wg, time.Second)

		select {
		case <-stopped:
		default:
			require.Fail(t, "waitForShutdown returned before the worker finished")
		}
	})
}

func Test__WaitForShutdown(t *testing.T) {
	t.Run("returns once every worker returns", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(10 * time.Millisecond)
		}()

		start := time.Now()
		waitForShutdown(&wg, time.Second)
		assert.Less(t, time.Since(start), time.Second)
	})

	t.Run("gives up when a worker does not return", func(t *testing.T) {
		var wg sync.WaitGroup
		release := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release
		}()
		defer close(release)

		start := time.Now()
		waitForShutdown(&wg, 50*time.Millisecond)
		elapsed := time.Since(start)

		//
		// A stuck worker must not hold the process open past the deadline.
		//
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
		assert.Less(t, elapsed, time.Second)
	})
}

func Test__StartConsumer(t *testing.T) {
	t.Run("keeps calling stop until the consumer returns", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup

		//
		// Mirrors tackle: Stop does nothing until the consumer reaches the
		// listening state, so the first Stop after cancellation is discarded.
		//
		var mu sync.Mutex
		listening := false
		stopCalls := 0
		release := make(chan struct{})

		start := func(startCtx context.Context) error {
			<-startCtx.Done()

			//
			// Stand in for tackle's connect(), which does network I/O before it
			// marks the consumer as listening. Every Stop that lands in this
			// window is discarded.
			//
			time.Sleep(4 * stopRetryInterval)

			mu.Lock()
			listening = true
			mu.Unlock()

			<-release
			return nil
		}

		stop := func() {
			mu.Lock()
			defer mu.Unlock()

			stopCalls++
			if !listening {
				return
			}

			select {
			case <-release:
			default:
				close(release)
			}
		}

		startConsumer(ctx, &wg, "test consumer", start, stop)
		cancel()
		waitForShutdown(&wg, 5*time.Second)

		mu.Lock()
		defer mu.Unlock()
		assert.Greater(t, stopCalls, 1, "a single lost stop must not strand the consumer")
	})

	t.Run("does not call stop when the consumer returns on its own", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		stopped := false

		startConsumer(ctx, &wg, "test consumer",
			func(context.Context) error { return nil },
			func() { stopped = true },
		)

		waitForShutdown(&wg, 5*time.Second)
		assert.False(t, stopped, "the watcher must exit when the consumer already returned")
	})
}
