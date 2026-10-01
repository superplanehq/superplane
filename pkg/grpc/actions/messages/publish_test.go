package messages

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish_UnreachableBrokerDoesNotLogConnectLine(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:1/")

	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() {
		log.SetOutput(previous)
	})

	err := Publish("superplane.canvas-exchange", "test.routing", []byte("body"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "RABBITMQ_URL not set")
	assert.NotContains(t, output.String(), "TACKLE: Connecting...")
}
