package logging

import (
	"bytes"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTackleLogger(t *testing.T) {
	logger := log.New()
	var buffer bytes.Buffer
	logger.SetOutput(&buffer)
	logger.SetLevel(log.DebugLevel)
	logger.SetFormatter(&log.TextFormatter{DisableTimestamp: true})

	tackle := NewTackleLogger(log.NewEntry(logger))

	tackle.Infof("hello %s", "world")
	require.Contains(t, buffer.String(), "hello world")
	assert.Contains(t, buffer.String(), "level=info")

	buffer.Reset()
	tackle.Errorf("boom %d", 42)
	require.Contains(t, buffer.String(), "boom 42")
	assert.Contains(t, buffer.String(), "level=error")
}
