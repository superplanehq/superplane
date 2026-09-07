package sentry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Sentry__Sync_hostedPending(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	defer withSentryIntakeEnabledForTest(func(string) bool { return true })()

	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Configuration: map[string]any{
			"setupReturnPath": "/onboarding",
		},
	}

	err := (&Sentry{}).Sync(core.SyncContext{
		Integration:    integrationCtx,
		Configuration:  integrationCtx.Configuration,
		BaseURL:        "https://app.example.com",
		OrganizationID: "org-1",
		ActorUserID:    "user-1",
	})
	require.NoError(t, err)
	require.NotNil(t, integrationCtx.BrowserAction)
	assert.Equal(t, "GET", integrationCtx.BrowserAction.Method)
	assert.Equal(
		t,
		"https://app.example.com/api/v1/sentry/app/start?integration=8f5fbc57-2738-409a-a6f8-af65c2de733c",
		integrationCtx.BrowserAction.URL,
	)

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.True(t, metadata.HostedApp)
	assert.Equal(t, "user-1", metadata.StartedByUserID)
	assert.Equal(t, "/onboarding", metadata.SetupReturnPath)
	assert.NotEmpty(t, metadata.State)
	assert.Empty(t, metadata.InstallationID)
}

func Test__Sentry__Sync_hostedKeepsPendingState(t *testing.T) {
	t.Setenv(config.EnvSentryAppClientID, "client-id")
	t.Setenv(config.EnvSentryAppClientSecret, "client-secret")
	t.Setenv(config.EnvSentryAppSlug, "superplane")
	defer withSentryIntakeEnabledForTest(func(string) bool { return true })()

	integrationCtx := &contexts.IntegrationContext{
		IntegrationID: "8f5fbc57-2738-409a-a6f8-af65c2de733c",
		Metadata: Metadata{
			HostedApp:       true,
			State:           "existing-nonce",
			StartedByUserID: "user-1",
			SetupReturnPath: "/onboarding",
		},
	}

	err := (&Sentry{}).Sync(core.SyncContext{
		Integration:    integrationCtx,
		BaseURL:        "https://app.example.com",
		OrganizationID: "org-1",
		ActorUserID:    "user-2",
	})
	require.NoError(t, err)
	metadata := integrationCtx.Metadata.(Metadata)
	assert.Equal(t, "existing-nonce", metadata.State)
	assert.Equal(t, "user-1", metadata.StartedByUserID)
	assert.Equal(t, "/onboarding", metadata.SetupReturnPath)
}
