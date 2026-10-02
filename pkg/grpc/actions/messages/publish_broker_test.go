package messages_test

import (
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	testconsumer "github.com/superplanehq/superplane/test/consumer"
)

func TestPublishCreatedDeliversToConsumer(t *testing.T) {
	amqpURL := os.Getenv("RABBITMQ_URL")
	if amqpURL == "" {
		t.Skip("RABBITMQ_URL not set")
	}

	c := testconsumer.NewForExchange(amqpURL, messages.CanvasExchange, messages.CanvasCreatedRoutingKey)
	c.Start()
	defer c.Stop()

	message := messages.NewCanvasCreatedMessage(uuid.NewString(), uuid.NewString())
	require.NoError(t, message.PublishCreated())
	require.True(t, c.HasReceivedMessage())
}
