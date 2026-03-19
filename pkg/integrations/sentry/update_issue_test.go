package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateIssue__Setup(t *testing.T) {
	component := &UpdateIssue{}

	t.Run("requires at least one update field", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
			},
		})

		require.ErrorContains(t, err, "at least one field to update")
	})

	t.Run("accepts status update", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
				"status":  "resolved",
			},
		})

		require.NoError(t, err)
	})
}

func Test__UpdateIssue__Execute(t *testing.T) {
	component := &UpdateIssue{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl": "https://sentry.io",
		},
		Metadata: Metadata{
			Organization: &OrganizationSummary{
				Slug: "example",
			},
		},
		Secrets: map[string]core.IntegrationSecret{
			OAuthAccessTokenSecret: {Name: OAuthAccessTokenSecret, Value: []byte("access-token")},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"123","status":"resolved","assignedTo":{"name":"Platform"}}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId":    "123",
			"status":     "resolved",
			"assignedTo": "platform",
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.issue", executionState.Type)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", httpCtx.Requests[0].URL.String())
}
