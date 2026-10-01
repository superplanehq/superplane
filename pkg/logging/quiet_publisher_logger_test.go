package logging

import (
	"bytes"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuietPublisherLogger_DropsInfoAndWritesErrors(t *testing.T) {
	previousOutput := log.StandardLogger().Out
	previousLevel := log.StandardLogger().GetLevel()
	buffer := &bytes.Buffer{}
	log.StandardLogger().SetOutput(buffer)
	log.StandardLogger().SetLevel(log.InfoLevel)
	t.Cleanup(func() {
		log.StandardLogger().SetOutput(previousOutput)
		log.StandardLogger().SetLevel(previousLevel)
	})

	logger := NewQuietPublisherLogger()
	logger.Infof("Connecting...")
	assert.Empty(t, buffer.String())

	logger.Errorf("publish failed: %s", "connection refused")

	logged := buffer.String()
	require.NotEmpty(t, logged)
	assert.Contains(t, logged, "publish failed: connection refused")
	assert.NotContains(t, logged, "Connecting")
}

func TestTackleLogger_WritesInfoAndError(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := log.New()
	logger.SetOutput(buffer)
	logger.SetLevel(log.InfoLevel)

	tackleLogger := NewTackleLogger(log.NewEntry(logger))
	tackleLogger.Infof("Connecting...")
	tackleLogger.Errorf("publish failed: %s", "connection refused")

	logged := buffer.String()
	assert.Contains(t, logged, "Connecting...")
	assert.Contains(t, logged, "publish failed: connection refused")
}
