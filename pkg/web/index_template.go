package web

import (
	"bytes"
	"html/template"
	"os"
	"strings"
)

type indexTemplateData struct {
	SentryDSN         string
	SentryEnvironment string
	AgentEnabled      bool
}

// agentEnabledFromEnv is true only when AGENT_ENABLED is the lowercase literal "yes" (after TrimSpace).
func agentEnabledFromEnv() bool {
	v := strings.TrimSpace(os.Getenv("AGENT_ENABLED"))
	return v == "yes"
}

func newIndexTemplateDataFromEnv() indexTemplateData {
	return indexTemplateData{
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: os.Getenv("SENTRY_ENVIRONMENT"),
		AgentEnabled:      agentEnabledFromEnv(),
	}
}

// RenderIndexTemplate renders the given index.html content as a Go template,
// injecting configuration (e.g. Sentry; AGENT_ENABLED must be lowercase "yes" to enable the agent UI).
// It returns an error if the template cannot be parsed or executed.
func RenderIndexTemplate(raw []byte) ([]byte, error) {
	tmpl, err := template.New("index.html").Parse(string(raw))
	if err != nil {
		return nil, err
	}

	data := newIndexTemplateDataFromEnv()

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
