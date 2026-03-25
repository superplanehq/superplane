package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetIssue__Setup(t *testing.T) {
	component := &GetIssue{}

	t.Run("stores issue metadata for a concrete issue id", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "123",
			},
			Metadata: metadata,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":   "https://sentry.io",
					"userToken": "user-token",
				},
				Metadata: Metadata{
					Organization: &OrganizationSummary{
						Slug: "example",
					},
				},
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout"}`),
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, GetIssueNodeMetadata{
			IssueTitle: "EXAMPLE-1 · Database timeout",
		}, metadata.Metadata)
	})

	t.Run("skips setup-time API validation for expressions", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		httpCtx := &contexts.HTTPContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId": "$['On Issue Event'].data.data.issue.id",
			},
			Metadata: metadata,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseUrl":   "https://sentry.io",
					"userToken": "user-token",
				},
				Metadata: Metadata{
					Organization: &OrganizationSummary{
						Slug: "example",
					},
				},
			},
			HTTP: httpCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, GetIssueNodeMetadata{}, metadata.Metadata)
		assert.Empty(t, httpCtx.Requests)
	})
}

func Test__GetIssue__Configuration(t *testing.T) {
	component := &GetIssue{}
	issueField := component.Configuration()[0]

	assert.Equal(t, configuration.FieldTypeIntegrationResource, issueField.Type)
	require.NotNil(t, issueField.TypeOptions)
	require.NotNil(t, issueField.TypeOptions.Resource)
	assert.Equal(t, ResourceTypeIssue, issueField.TypeOptions.Resource.Type)
}

func Test__GetIssue__Execute(t *testing.T) {
	component := &GetIssue{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout","count":"42","status":"unresolved","userCount":5,"assignedTo":{"name":"Platform"},"project":{"name":"Backend","slug":"backend"},"tags":[{"key":"environment","value":"production"}]}`),
			sentryMockResponse(http.StatusOK, `[{"id":"evt-1","eventID":"evt-1","title":"RuntimeError: Database timeout","dateCreated":"2026-03-25T10:00:00Z","tags":[{"key":"environment","value":"production"}]}]`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId": "123",
		},
		HTTP: httpCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":   "https://sentry.io",
				"userToken": "user-token",
			},
			Metadata: Metadata{
				Organization: &OrganizationSummary{
					Slug: "example",
				},
			},
		},
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.issue", executionState.Type)
	require.Len(t, httpCtx.Requests, 2)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/events/?limit=10", httpCtx.Requests[1].URL.String())
}
