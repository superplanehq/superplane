package messages

import (
	"bytes"
	stdlog "log"
	"testing"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublish_UnreachableBrokerDoesNotLogConnectLine(t *testing.T) {
	t.Setenv("RABBITMQ_URL", "amqp://guest:guest@127.0.0.1:1/")

	var stdlib bytes.Buffer
	previousStdlib := stdlog.Writer()
	stdlog.SetOutput(&stdlib)
	t.Cleanup(func() {
		stdlog.SetOutput(previousStdlib)
	})

	processLog := &bytes.Buffer{}
	previousOutput := logrus.StandardLogger().Out
	previousLevel := logrus.StandardLogger().GetLevel()
	logrus.StandardLogger().SetOutput(processLog)
	logrus.StandardLogger().SetLevel(logrus.InfoLevel)
	t.Cleanup(func() {
		logrus.StandardLogger().SetOutput(previousOutput)
		logrus.StandardLogger().SetLevel(previousLevel)
	})

	err := Publish("superplane.canvas-exchange", "test.routing", []byte("body"))
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "RABBITMQ_URL not set")

	logged := stdlib.String() + processLog.String()
	assert.NotContains(t, logged, "TACKLE: Connecting...")
	assert.NotContains(t, logged, "Connecting...")
}
