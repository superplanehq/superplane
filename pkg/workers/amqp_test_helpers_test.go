package workers

import (
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func purgeRabbitQueueEventually(t *testing.T, amqpURL, queueName string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := amqp.Dial(amqpURL)
		if err == nil {
			ch, chErr := conn.Channel()
			if chErr == nil {
				_, purgeErr := ch.QueuePurge(queueName, false)
				_ = ch.Close()
				_ = conn.Close()
				if purgeErr == nil {
					return
				}
			} else {
				_ = conn.Close()
			}
		}

		if time.Now().After(deadline) {
			require.FailNowf(t, "failed to purge rabbitmq queue", "queue=%q", queueName)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
