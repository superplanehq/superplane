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

func agentEnabled() bool {
	return strings.TrimSpace(os.Getenv("AGENT_ENABLED")) == "yes"
}

func newIndexTemplateDataFromEnv() indexTemplateData {
	return indexTemplateData{
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: os.Getenv("SENTRY_ENVIRONMENT"),
		AgentEnabled:      agentEnabled(),
	}
}

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
