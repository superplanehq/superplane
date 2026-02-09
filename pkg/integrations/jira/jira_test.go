package jira

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Jira__Sync(t *testing.T) {
	j := &Jira{}

	t.Run("no baseUrl -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "baseUrl is required")
	})

	t.Run("no email -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "",
				"apiToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "email is required")
	})

	t.Run("no apiToken -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			Integration:   appCtx,
		})

		require.ErrorContains(t, err, "apiToken is required")
	})

	t.Run("successful sync -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"accountId":"123"}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"id":"10000","key":"TEST","name":"Test Project"}]`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", appCtx.State)
	})

	t.Run("auth failure -> error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"message":"unauthorized"}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "invalid-token",
			},
		}

		err := j.Sync(core.SyncContext{
			Configuration: appCtx.Configuration,
			HTTP:          httpContext,
			Integration:   appCtx,
		})

		require.Error(t, err)
		assert.NotEqual(t, "ready", appCtx.State)
	})
}

func Test__Jira__CompareWebhookConfig(t *testing.T) {
	j := &Jira{}

	testCases := []struct {
		name        string
		configA     any
		configB     any
		expectEqual bool
		expectError bool
	}{
		{
			name: "identical configurations",
			configA: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			configB: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name: "different event types",
			configA: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			configB: WebhookConfiguration{
				EventType: "jira:issue_updated",
				Project:   "TEST",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "different projects",
			configA: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			configB: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "OTHER",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "both fields different",
			configA: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			configB: WebhookConfiguration{
				EventType: "jira:issue_updated",
				Project:   "OTHER",
			},
			expectEqual: false,
			expectError: false,
		},
		{
			name: "comparing map representations",
			configA: map[string]any{
				"eventType": "jira:issue_created",
				"project":   "TEST",
			},
			configB: map[string]any{
				"eventType": "jira:issue_created",
				"project":   "TEST",
			},
			expectEqual: true,
			expectError: false,
		},
		{
			name:    "invalid first configuration",
			configA: "invalid",
			configB: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			expectEqual: false,
			expectError: true,
		},
		{
			name: "invalid second configuration",
			configA: WebhookConfiguration{
				EventType: "jira:issue_created",
				Project:   "TEST",
			},
			configB:     "invalid",
			expectEqual: false,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			equal, err := j.CompareWebhookConfig(tc.configA, tc.configB)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.expectEqual, equal)
		})
	}
}

func Test__Jira__HandleAction(t *testing.T) {
	j := &Jira{}

	t.Run("listWebhooks -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"startAt":0,"maxResults":50,"total":1,"values":[{"id":123,"jqlFilter":"project = TEST","events":["jira:issue_created"]}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "listWebhooks",
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/webhook")
	})

	t.Run("deleteWebhook -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "deleteWebhook",
			HTTP:        httpContext,
			Integration: appCtx,
			Parameters:  map[string]any{"webhookId": float64(123)},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("deleteWebhook -> missing webhookId", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "deleteWebhook",
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Parameters:  map[string]any{},
		})

		require.ErrorContains(t, err, "webhookId parameter is required")
	})

	t.Run("deleteWebhook -> invalid parameters type", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "deleteWebhook",
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
			Parameters:  "invalid",
		})

		require.ErrorContains(t, err, "invalid parameters")
	})

	t.Run("deleteAllWebhooks -> success with no webhooks", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"startAt":0,"maxResults":50,"total":0,"values":[]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "deleteAllWebhooks",
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
	})

	t.Run("deleteAllWebhooks -> success with existing webhooks", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"startAt":0,"maxResults":50,"total":2,"values":[{"id":1},{"id":2}]}`)),
				},
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "deleteAllWebhooks",
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		// First request: list webhooks, second request: delete webhooks
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	})

	t.Run("getFailedWebhooks -> success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"values":[{"id":"1","url":"http://example.com","failureReason":"timeout","latestFailureTime":"2024-01-01T00:00:00Z"}]}`)),
				},
			},
		}

		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "getFailedWebhooks",
			HTTP:        httpContext,
			Integration: appCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/rest/api/3/webhook/failed")
	})

	t.Run("unknown action -> error", func(t *testing.T) {
		appCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseUrl":  "https://test.atlassian.net",
				"email":    "test@example.com",
				"apiToken": "test-token",
			},
		}

		err := j.HandleAction(core.IntegrationActionContext{
			Name:        "unknownAction",
			HTTP:        &contexts.HTTPContext{},
			Integration: appCtx,
		})

		require.ErrorContains(t, err, "unknown action: unknownAction")
	})
}

func Test__Jira__Actions(t *testing.T) {
	j := &Jira{}
	actions := j.Actions()

	require.Len(t, actions, 4)

	actionNames := make([]string, len(actions))
	for i, a := range actions {
		actionNames[i] = a.Name
	}

	assert.Contains(t, actionNames, "listWebhooks")
	assert.Contains(t, actionNames, "deleteWebhook")
	assert.Contains(t, actionNames, "deleteAllWebhooks")
	assert.Contains(t, actionNames, "getFailedWebhooks")

	for _, a := range actions {
		assert.True(t, a.UserAccessible, fmt.Sprintf("%s should be user accessible", a.Name))
	}
}

func Test__Jira__IntegrationInfo(t *testing.T) {
	j := &Jira{}

	assert.Equal(t, "jira", j.Name())
	assert.Equal(t, "Jira", j.Label())
	assert.Equal(t, "jira", j.Icon())
	assert.NotEmpty(t, j.Description())
}

func Test__Jira__Components(t *testing.T) {
	j := &Jira{}
	components := j.Components()

	require.Len(t, components, 3)

	componentNames := make([]string, len(components))
	for i, c := range components {
		componentNames[i] = c.Name()
	}

	assert.Contains(t, componentNames, "jira.createIssue")
	assert.Contains(t, componentNames, "jira.listWebhooks")
	assert.Contains(t, componentNames, "jira.deleteWebhooks")
}

func Test__Jira__Triggers(t *testing.T) {
	j := &Jira{}
	triggers := j.Triggers()

	require.Len(t, triggers, 1)
	assert.Equal(t, "jira.onIssueCreated", triggers[0].Name())
}
