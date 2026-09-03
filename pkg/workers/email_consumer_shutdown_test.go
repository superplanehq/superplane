package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The consumers reconnect in a loop. Before they took a context, a cancelled
// process still reconnected and accepted new deliveries during the drain, and
// Start never returned, so nothing could wait for them.
func Test__EmailConsumersStopOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("magic code consumer", func(t *testing.T) {
		consumer := NewMagicCodeEmailConsumer("amqp://127.0.0.1:1", nil, "https://app.superplane.com")
		requireReturnsBefore(t, time.Second, func() error { return consumer.Start(ctx) })
	})

	t.Run("factory notification consumer", func(t *testing.T) {
		consumer := NewFactoryNotificationConsumer("amqp://127.0.0.1:1", nil, "https://app.superplane.com")
		requireReturnsBefore(t, time.Second, func() error { return consumer.Start(ctx) })
	})
}

// requireReturnsBefore fails if start has not returned when timeout expires.
// The unreachable broker address makes the test independent of RabbitMQ.
func requireReturnsBefore(t *testing.T, timeout time.Duration, start func() error) {
	t.Helper()

	done := make(chan error, 1)
	go func() { done <- start() }()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(timeout):
		require.Fail(t, "Start did not return for a cancelled context")
	}
}
