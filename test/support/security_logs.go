package support

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// CaptureSecurityLogs captures the process output at real operation boundaries.
func CaptureSecurityLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	logger := log.StandardLogger()
	output, formatter := logger.Out, logger.Formatter
	buffer := &bytes.Buffer{}
	logger.SetOutput(buffer)
	logger.SetFormatter(&log.JSONFormatter{})
	t.Cleanup(func() { logger.SetOutput(output); logger.SetFormatter(formatter) })
	return buffer
}

func SecurityEvents(t *testing.T, buffer *bytes.Buffer) []map[string]any {
	t.Helper()
	events := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		if record["type"] == "security_event" {
			events = append(events, record)
		}
	}
	return events
}
