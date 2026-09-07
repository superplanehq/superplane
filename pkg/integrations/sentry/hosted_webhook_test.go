package sentry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__handleWebhook_hostedIssueFanout(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")

	body := `{"action":"created","installation":{"uuid":"install-1"},"data":{"issue":{"id":"9"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/sentry/app/webhook", strings.NewReader(body))
	req.Header.Set("Sentry-Hook-Signature", computeWebhookSignature("client-secret", []byte(body)))
	req.Header.Set("Sentry-Hook-Resource", "issue")
	rec := httptest.NewRecorder()
	subscription := contexts.Subscription{Configuration: SubscriptionConfiguration{Resources: []string{"issue"}}}
	integration := &contexts.IntegrationContext{
		Metadata:      Metadata{HostedApp: true, InstallationID: "install-1"},
		Subscriptions: []contexts.Subscription{subscription},
	}

	(&Sentry{}).HandleRequest(core.HTTPRequestContext{
		Logger:      logrus.NewEntry(logrus.New()),
		Request:     req,
		Response:    rec,
		Integration: integration,
	})

	assert.Equal(t, http.StatusOK, rec.Code)
}
