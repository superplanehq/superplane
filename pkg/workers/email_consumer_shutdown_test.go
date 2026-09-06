package workers

import (
	"context"
	"testing"
	"time"

	tackle "github.com/renderedtext/go-tackle"
	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
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

// A deliberate Stop during shutdown makes tackle's Start return nil, which the
// reconnect path would otherwise report as a dropped connection. A deploy would
// then log a reconnect warning for every consumer, every time.
func Test__EmailConsumersDoNotWarnAboutReconnectOnShutdown(t *testing.T) {
	amqpURL, err := config.RabbitMQURL()
	require.NoError(t, err)

	magicCode := NewMagicCodeEmailConsumer(amqpURL, nil, "https://app.superplane.com")
	factoryNotification := NewFactoryNotificationConsumer(amqpURL, nil, "https://app.superplane.com")

	consumers := []shutdownTestConsumer{
		{"magic code", magicCode.Start, magicCode.Stop, magicCode.Consumer},
		{"factory notification", factoryNotification.Start, factoryNotification.Stop, factoryNotification.Consumer},
	}

	for _, consumer := range consumers {
		t.Run(consumer.name, func(t *testing.T) {
			hook := logtest.NewGlobal()
			defer hook.Reset()

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan error, 1)
			go func() { done <- consumer.start(ctx) }()

			//
			// Wait until the consumer is really listening, so Stop closes a live
			// connection and Start returns nil, which is the shutdown path under test.
			//
			require.Eventually(t, func() bool {
				return consumer.consumer.State == tackle.StateListening
			}, 10*time.Second, 20*time.Millisecond, "consumer never started listening")

			hook.Reset()
			cancel()
			consumer.stop()

			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(10 * time.Second):
				require.Fail(t, "Start did not return after Stop")
			}

			for _, entry := range hook.AllEntries() {
				assert.NotContains(t, entry.Message, "reconnecting", "a deliberate stop is not a dropped connection")
				assert.NotEqual(t, log.ErrorLevel, entry.Level, "a deliberate stop is not an error")
			}
		})
	}
}

// shutdownTestConsumer describes one email consumer for the shutdown tests, so
// both consumers run the same assertions.
type shutdownTestConsumer struct {
	name     string
	start    func(context.Context) error
	stop     func()
	consumer *tackle.Consumer
}
