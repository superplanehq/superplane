package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudLoggingFormatter_SeverityAndMessage(t *testing.T) {
	formatter := NewCloudLoggingFormatter()
	cases := []struct {
		level    log.Level
		severity string
	}{
		{level: log.DebugLevel, severity: "DEBUG"},
		{level: log.InfoLevel, severity: "INFO"},
		{level: log.WarnLevel, severity: "WARNING"},
		{level: log.ErrorLevel, severity: "ERROR"},
		{level: log.FatalLevel, severity: "CRITICAL"},
		{level: log.PanicLevel, severity: "ALERT"},
		{level: log.TraceLevel, severity: "DEBUG"},
	}

	for _, tc := range cases {
		t.Run(tc.level.String(), func(t *testing.T) {
			entry := &log.Entry{
				Time:    time.Date(2026, 9, 30, 8, 11, 12, 233707649, time.UTC),
				Level:   tc.level,
				Message: "handled request",
				Data: log.Fields{
					"status": 200,
				},
			}

			line, err := formatter.Format(entry)
			require.NoError(t, err)

			payload := map[string]any{}
			require.NoError(t, json.Unmarshal(line, &payload))
			assert.Equal(t, tc.severity, payload["severity"])
			assert.Equal(t, "handled request", payload["message"])
			assert.EqualValues(t, 200, payload["status"])
			_, hasLevel := payload["level"]
			assert.False(t, hasLevel)
			_, hasMsg := payload["msg"]
			assert.False(t, hasMsg)

			parsed, err := time.Parse(time.RFC3339Nano, payload["time"].(string))
			require.NoError(t, err)
			assert.True(t, entry.Time.Equal(parsed))
		})
	}
}

func TestConfigureProcessLogger_JSONFormatUsesCloudLoggingFormatter(t *testing.T) {
	restoreProcessLogger(t)
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("APP_ENV", "production")

	ConfigureProcessLogger()

	_, isCloudLogging := log.StandardLogger().Formatter.(*CloudLoggingFormatter)
	assert.True(t, isCloudLogging)
	assert.Equal(t, os.Stdout, log.StandardLogger().Out)
}

func TestConfigureProcessLogger_WithoutJSONKeepsTextFormatter(t *testing.T) {
	restoreProcessLogger(t)
	buffer := &bytes.Buffer{}
	log.SetOutput(buffer)
	t.Setenv("LOG_FORMAT", "")
	t.Setenv("APP_ENV", "production")

	ConfigureProcessLogger()

	_, isText := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, isText)
	assert.Equal(t, buffer, log.StandardLogger().Out)
}

func restoreProcessLogger(t *testing.T) {
	t.Helper()

	previousFormatter := log.StandardLogger().Formatter
	previousOutput := log.StandardLogger().Out
	t.Cleanup(func() {
		log.SetFormatter(previousFormatter)
		log.SetOutput(previousOutput)
	})
}
