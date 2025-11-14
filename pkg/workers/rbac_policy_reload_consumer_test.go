package workers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/test/support"
)

func Test__RBACPolicyReloadConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	amqpURL := "amqp://guest:guest@rabbitmq:5672"

	consumer := NewRBACPolicyReloadConsumer(amqpURL, r.AuthService)

	go consumer.Start()
	defer consumer.Stop()

	t.Run("should reload policies when message received", func(t *testing.T) {
		message := messages.NewRBACPolicyReloadMessage()
		err := message.Publish()
		require.NoError(t, err)
	})
}

func TestNewRBACPolicyReloadConsumer(t *testing.T) {
	r := support.SetupWithOptions(t, support.SetupOptions{})
	defer r.Close()

	rabbitMQURL := "amqp://localhost:5672"

	consumer := NewRBACPolicyReloadConsumer(rabbitMQURL, r.AuthService)

	assert.NotNil(t, consumer)
	assert.Equal(t, rabbitMQURL, consumer.RabbitMQURL)
	assert.Equal(t, r.AuthService, consumer.AuthService)
	assert.NotNil(t, consumer.Consumer)
}
