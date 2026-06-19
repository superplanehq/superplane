package web

import (
	"bytes"
	"html/template"
	"os"
	"strings"
)

type indexTemplateData struct {
	SentryDSN                     string
	SentryEnvironment             string
	PostHogKey                    string
	Dash0WebOTLPEndpoint          string
	Dash0WebAuthToken             string
	Dash0WebServiceName           string
	Dash0WebEnvironment           string
	SignupWaitlistHubSpotPortalID string
	SignupWaitlistHubSpotFormID   string
	SignupWaitlistHubSpotRegion   string
}

func dash0WebEnvironment() string {
	if environment := strings.TrimSpace(os.Getenv("DASH0_WEB_ENVIRONMENT")); environment != "" {
		return environment
	}

	if environment := strings.TrimSpace(os.Getenv("SENTRY_ENVIRONMENT")); environment != "" {
		return environment
	}

	return strings.TrimSpace(os.Getenv("APP_ENV"))
}

func dash0WebServiceName() string {
	if serviceName := strings.TrimSpace(os.Getenv("DASH0_WEB_SERVICE_NAME")); serviceName != "" {
		return serviceName
	}

	return "superplane-web"
}

func newIndexTemplateDataFromEnv() indexTemplateData {
	return indexTemplateData{
		SentryDSN:                     os.Getenv("SENTRY_DSN"),
		SentryEnvironment:             os.Getenv("SENTRY_ENVIRONMENT"),
		PostHogKey:                    os.Getenv("POSTHOG_KEY"),
		Dash0WebOTLPEndpoint:          os.Getenv("DASH0_WEB_OTLP_ENDPOINT"),
		Dash0WebAuthToken:             os.Getenv("DASH0_WEB_AUTH_TOKEN"),
		Dash0WebServiceName:           dash0WebServiceName(),
		Dash0WebEnvironment:           dash0WebEnvironment(),
		SignupWaitlistHubSpotPortalID: os.Getenv("SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID"),
		SignupWaitlistHubSpotFormID:   os.Getenv("SIGNUP_WAITLIST_HUBSPOT_FORM_ID"),
		SignupWaitlistHubSpotRegion:   os.Getenv("SIGNUP_WAITLIST_HUBSPOT_REGION"),
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
