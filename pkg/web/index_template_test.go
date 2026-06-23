package web

import (
	"strings"
	"testing"
)

func TestRenderIndexTemplateIncludesDash0Config(t *testing.T) {
	t.Setenv("DASH0_WEB_OTLP_ENDPOINT", "https://ingress.us-west-2.aws.dash0.com:4318")
	t.Setenv("DASH0_WEB_AUTH_TOKEN", "dash0-token")
	t.Setenv("DASH0_WEB_SERVICE_NAME", "superplane-prod")
	t.Setenv("DASH0_WEB_ENVIRONMENT", "production")

	raw := []byte(`<script>
window.SUPERPLANE_DASH0_OTLP_ENDPOINT = "{{ .Dash0WebOTLPEndpoint }}";
window.SUPERPLANE_DASH0_AUTH_TOKEN = "{{ .Dash0WebAuthToken }}";
window.SUPERPLANE_DASH0_SERVICE_NAME = "{{ .Dash0WebServiceName }}";
window.SUPERPLANE_DASH0_ENVIRONMENT = "{{ .Dash0WebEnvironment }}";
</script>`)

	rendered, err := RenderIndexTemplate(raw)
	if err != nil {
		t.Fatalf("RenderIndexTemplate() error = %v", err)
	}

	body := string(rendered)
	for _, want := range []string{
		`window.SUPERPLANE_DASH0_OTLP_ENDPOINT = "https:\/\/ingress.us-west-2.aws.dash0.com:4318"`,
		`window.SUPERPLANE_DASH0_AUTH_TOKEN = "dash0-token"`,
		`window.SUPERPLANE_DASH0_SERVICE_NAME = "superplane-prod"`,
		`window.SUPERPLANE_DASH0_ENVIRONMENT = "production"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected rendered template to contain %q, got %q", want, body)
		}
	}
}

func TestRenderIndexTemplateIncludesSignupWaitlistHubSpotConfig(t *testing.T) {
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID", "portal-1")
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_FORM_ID", "form-1")
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_REGION", "eu1")

	raw := []byte(`<script>
window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "{{ .SignupWaitlistHubSpotPortalID }}";
window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "{{ .SignupWaitlistHubSpotFormID }}";
window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION = "{{ .SignupWaitlistHubSpotRegion }}";
</script>`)

	rendered, err := RenderIndexTemplate(raw)
	if err != nil {
		t.Fatalf("RenderIndexTemplate() error = %v", err)
	}

	body := string(rendered)
	for _, want := range []string{
		`window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID = "portal-1"`,
		`window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_FORM_ID = "form-1"`,
		`window.SUPERPLANE_SIGNUP_WAITLIST_HUBSPOT_REGION = "eu1"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected rendered template to contain %q, got %q", want, body)
		}
	}
}

func TestDash0WebServiceNameDefaults(t *testing.T) {
	t.Setenv("DASH0_WEB_SERVICE_NAME", "")

	if got := dash0WebServiceName(); got != "superplane-web" {
		t.Fatalf("dash0WebServiceName() = %q, want superplane-web", got)
	}
}

func TestDash0WebEnvironmentFallback(t *testing.T) {
	t.Setenv("DASH0_WEB_ENVIRONMENT", "")
	t.Setenv("SENTRY_ENVIRONMENT", "")
	t.Setenv("APP_ENV", "staging")

	if got := dash0WebEnvironment(); got != "staging" {
		t.Fatalf("dash0WebEnvironment() = %q, want staging", got)
	}
}

func TestNewIndexTemplateDataFromEnv(t *testing.T) {
	t.Setenv("DASH0_WEB_OTLP_ENDPOINT", "https://example.test:4318")
	t.Setenv("DASH0_WEB_AUTH_TOKEN", "token")
	t.Setenv("DASH0_WEB_SERVICE_NAME", "")
	t.Setenv("DASH0_WEB_ENVIRONMENT", "development")
	t.Setenv("SENTRY_DSN", "")
	t.Setenv("SENTRY_ENVIRONMENT", "")
	t.Setenv("POSTHOG_KEY", "")
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_PORTAL_ID", "portal-1")
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_FORM_ID", "form-1")
	t.Setenv("SIGNUP_WAITLIST_HUBSPOT_REGION", "eu1")

	data := newIndexTemplateDataFromEnv()
	if data.Dash0WebOTLPEndpoint != "https://example.test:4318" {
		t.Fatalf("Dash0WebOTLPEndpoint = %q", data.Dash0WebOTLPEndpoint)
	}
	if data.Dash0WebAuthToken != "token" {
		t.Fatalf("Dash0WebAuthToken = %q", data.Dash0WebAuthToken)
	}
	if data.Dash0WebServiceName != "superplane-web" {
		t.Fatalf("Dash0WebServiceName = %q", data.Dash0WebServiceName)
	}
	if data.Dash0WebEnvironment != "development" {
		t.Fatalf("Dash0WebEnvironment = %q", data.Dash0WebEnvironment)
	}
	if data.SignupWaitlistHubSpotPortalID != "portal-1" {
		t.Fatalf("SignupWaitlistHubSpotPortalID = %q", data.SignupWaitlistHubSpotPortalID)
	}
	if data.SignupWaitlistHubSpotFormID != "form-1" {
		t.Fatalf("SignupWaitlistHubSpotFormID = %q", data.SignupWaitlistHubSpotFormID)
	}
	if data.SignupWaitlistHubSpotRegion != "eu1" {
		t.Fatalf("SignupWaitlistHubSpotRegion = %q", data.SignupWaitlistHubSpotRegion)
	}
}
