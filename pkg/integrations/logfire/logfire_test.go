package logfire

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Logfire__Sync__MissingAPIKey(t *testing.T) {
	integration := &Logfire{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          &contexts.HTTPContext{},
		Integration:   integrationCtx,
	})

	require.ErrorContains(t, err, "apiKey is required")
}

func Test__Logfire__Sync__Success(t *testing.T) {
	integration := &Logfire{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`[{"id":"proj_123","organization_name":"acme-org","project_name":"backend"}]`,
				)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"token":"lf_read_token_123"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"columns":[],"rows":[]}`)),
			},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"apiKey": "lf_api_key_123",
		},
		Metadata: map[string]any{
			"supportsWebhookSetup": true,
		},
		Secrets: map[string]core.IntegrationSecret{},
	}

	err := integration.Sync(core.SyncContext{
		Configuration: integrationCtx.Configuration,
		HTTP:          httpCtx,
		Integration:   integrationCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, "ready", integrationCtx.State)

	secret, ok := integrationCtx.Secrets[readTokenSecretName]
	require.True(t, ok)
	assert.Equal(t, "lf_read_token_123", string(secret.Value))

	metadata, ok := integrationCtx.Metadata.(Metadata)
	require.True(t, ok)
	assert.Equal(t, "acme-org", metadata.ExternalOrganizationID)
	assert.Equal(t, "proj_123", metadata.ExternalProjectID)
	assert.True(t, metadata.SupportsWebhookSetup)
	assert.True(t, metadata.SupportsQueryAPI)
}
