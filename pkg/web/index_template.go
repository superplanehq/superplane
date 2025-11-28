package web

import (
	"bytes"
	"html/template"
	"os"
)

type indexTemplateData struct {
	SentryDSN         string
	SentryEnvironment string
}

func newIndexTemplateDataFromEnv() indexTemplateData {
	return indexTemplateData{
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		SentryEnvironment: os.Getenv("SENTRY_ENVIRONMENT"),
	}
}

// RenderIndexTemplate renders the given index.html content as a Go template,
// injecting configuration (e.g. Sentry settings) from environment variables.
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
