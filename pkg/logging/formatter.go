package logging

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// CloudLoggingFormatter writes one JSON object per line with the field names
// Cloud Logging reads from stdout: severity, message, and time.
type CloudLoggingFormatter struct {
	json log.JSONFormatter
}

// ConfigureProcessLogger selects the process log format.
// LOG_FORMAT=json writes Cloud Logging JSON to stdout. Any other value keeps
// the text formatter, so worker processes stay unchanged.
func ConfigureProcessLogger() {
	if os.Getenv("LOG_FORMAT") == "json" {
		log.SetFormatter(NewCloudLoggingFormatter())
		log.SetOutput(os.Stdout)
		return
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "development" || appEnv == "test" {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:   false,
			TimestampFormat: time.Stamp,
		})
		return
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.StampMilli,
	})
}

func NewCloudLoggingFormatter() *CloudLoggingFormatter {
	return &CloudLoggingFormatter{
		json: log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
			FieldMap: log.FieldMap{
				log.FieldKeyTime:  "time",
				log.FieldKeyLevel: "severity",
				log.FieldKeyMsg:   "message",
			},
		},
	}
}

func (f *CloudLoggingFormatter) Format(entry *log.Entry) ([]byte, error) {
	line, err := f.json.Format(entry)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(line, &payload); err != nil {
		return nil, err
	}

	if severity, ok := payload["severity"].(string); ok {
		payload["severity"] = cloudLoggingSeverity(severity)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return append(encoded, '\n'), nil
}

// cloudLoggingSeverity maps a logrus level name to a Cloud Logging severity.
// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
func cloudLoggingSeverity(level string) string {
	switch strings.ToLower(level) {
	case "warning":
		return "WARNING"
	case "fatal":
		return "CRITICAL"
	case "panic":
		return "ALERT"
	case "trace":
		return "DEBUG"
	default:
		return strings.ToUpper(level)
	}
}
