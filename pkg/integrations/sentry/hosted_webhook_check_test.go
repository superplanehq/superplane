package sentry

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestMarkHostedInstallReady(t *testing.T) {
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	t.Setenv(config.EnvSentryAppClientID, "cid")
	t.Setenv(config.EnvSentryAppClientSecret, "csecret")
	t.Setenv(config.EnvSentryAppAPIToken, "owner-token")

	expected := "https://hooks.example.com/api/v1/sentry/app/webhook"

	t.Run("missing issue event sets an error", func(t *testing.T) {
		integration := &contexts.IntegrationContext{}
		sentry := &Sentry{}
		err := sentry.markHostedInstallReady(core.SyncContext{
			Logger:          logrus.NewEntry(logrus.New()),
			WebhooksBaseURL: "https://hooks.example.com",
			HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"webhookUrl":"https://wrong.example/hook","events":["error"]}`)),
				Header:     make(http.Header),
			}}},
			Integration: integration,
		})
		require.NoError(t, err)
		assert.Equal(t, "error", integration.State)
		assert.Contains(t, integration.StateDescription, expected)
		assert.Contains(t, integration.StateDescription, "issue event")
	})

	t.Run("matching webhook stays ready", func(t *testing.T) {
		integration := &contexts.IntegrationContext{}
		sentry := &Sentry{}
		err := sentry.markHostedInstallReady(core.SyncContext{
			Logger:          logrus.NewEntry(logrus.New()),
			WebhooksBaseURL: "https://hooks.example.com",
			HTTP: &contexts.HTTPContext{Responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"webhookUrl":"https://hooks.example.com/api/v1/sentry/app/webhook","events":["issue"]}`,
				)),
				Header: make(http.Header),
			}}},
			Integration: integration,
		})
		require.NoError(t, err)
		assert.Equal(t, "ready", integration.State)
		assert.Empty(t, integration.StateDescription)
	})
}
