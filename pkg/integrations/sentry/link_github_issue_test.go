package sentry

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__LinkGitHubIssue__Setup(t *testing.T) {
	component := &LinkGitHubIssue{}

	t.Run("requires issueId", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"githubIntegrationId": "456",
				"repo":                "example-org/example-repo",
				"externalIssue":       "42",
			},
		})

		require.ErrorContains(t, err, "issueId is required")
	})

	t.Run("requires githubIntegrationId", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId":       "123",
				"repo":          "example-org/example-repo",
				"externalIssue": "42",
			},
		})

		require.ErrorContains(t, err, "githubIntegrationId is required")
	})

	t.Run("requires repo in owner/repo format", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId":             "123",
				"githubIntegrationId": "456",
				"repo":                "example-repo",
				"externalIssue":       "42",
			},
		})

		require.ErrorContains(t, err, "repo must be in owner/repo format")
	})

	t.Run("stores metadata for configured issue and integration", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"issueId":             "123",
				"githubIntegrationId": "456",
				"repo":                "example-org/example-repo",
				"externalIssue":       "42",
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
					sentryMockResponse(http.StatusOK, `{"id":"123","shortId":"EXAMPLE-1","title":"RuntimeError: Database timeout","project":{"slug":"backend","name":"Backend"}}`),
					sentryMockResponse(http.StatusOK, `[{"id":"456","name":"GitHub","status":"active","provider":{"key":"github","name":"GitHub"}}]`),
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, LinkGitHubIssueNodeMetadata{
			IssueTitle:             "EXAMPLE-1 · Database timeout",
			GitHubIntegrationLabel: "GitHub",
			ExternalIssueLabel:     "example-org/example-repo#42",
		}, metadata.Metadata)
	})

	t.Run("uses issue and github integration resources", func(t *testing.T) {
		fields := component.Configuration()

		assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
		require.NotNil(t, fields[0].TypeOptions)
		require.NotNil(t, fields[0].TypeOptions.Resource)
		assert.Equal(t, ResourceTypeIssue, fields[0].TypeOptions.Resource.Type)

		assert.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
		require.NotNil(t, fields[1].TypeOptions)
		require.NotNil(t, fields[1].TypeOptions.Resource)
		assert.Equal(t, ResourceTypeGitHubIntegration, fields[1].TypeOptions.Resource.Type)
	})
}

func Test__LinkGitHubIssue__Execute(t *testing.T) {
	component := &LinkGitHubIssue{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": "user-token",
		},
		Metadata: Metadata{
			Organization: &OrganizationSummary{
				Slug: "example",
			},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, `{"id":5,"key":"42","url":"https://github.com/example-org/example-repo/issues/42","integrationId":456,"displayName":"example-org/example-repo#42"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId":             "123",
			"githubIntegrationId": "456",
			"repo":                "example-org/example-repo",
			"externalIssue":       "42",
			"comment":             "Linked from SuperPlane",
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)
	assert.Equal(t, "sentry.externalIssue", executionState.Type)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, "https://sentry.io/api/0/organizations/example/issues/123/integrations/456/", httpCtx.Requests[0].URL.String())
	assert.Equal(t, http.MethodPut, httpCtx.Requests[0].Method)

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "example-org/example-repo", requestBody["repo"])
	assert.EqualValues(t, 42, requestBody["externalIssue"])
	assert.Equal(t, "Linked from SuperPlane", requestBody["comment"])
}

func Test__LinkGitHubIssue__Execute_AcceptsNonNumericExternalIssue(t *testing.T) {
	component := &LinkGitHubIssue{}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"baseUrl":   "https://sentry.io",
			"userToken": "user-token",
		},
		Metadata: Metadata{
			Organization: &OrganizationSummary{
				Slug: "example",
			},
		},
	}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusCreated, `{"id":5,"key":"SCRUM-24","url":"https://example.atlassian.net/browse/SCRUM-24","integrationId":456,"displayName":"SCRUM-24"}`),
		},
	}
	executionState := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"issueId":             "123",
			"githubIntegrationId": "456",
			"repo":                "example-org/example-repo",
			"externalIssue":       "SCRUM-24",
		},
		HTTP:           httpCtx,
		Integration:    integrationCtx,
		ExecutionState: executionState,
	})

	require.NoError(t, err)
	assert.True(t, executionState.Passed)

	body, readErr := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, readErr)

	requestBody := map[string]any{}
	require.NoError(t, json.Unmarshal(body, &requestBody))
	assert.Equal(t, "SCRUM-24", requestBody["externalIssue"])
}
