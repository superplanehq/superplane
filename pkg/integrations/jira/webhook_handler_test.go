package jira

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

func Test__JiraWebhookHandler__Setup(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("registers webhook with jira and returns ids", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"createdWebhookId":99}]}`)),
				},
			},
		}

		webhook := &contexts.WebhookContext{
			ID:  "wh-1",
			URL: "https://hook.example.com/wh-1",
			Configuration: WebhookConfiguration{
				Events:    []string{JiraEventIssueCreated, JiraEventIssueUpdated},
				JQLFilter: "project = TEST",
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook:     webhook,
		})

		require.NoError(t, err)
		wm, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, []int{99}, wm.IDs)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/ex/jira/cloud-123/rest/api/3/webhook")
	})

	t.Run("missing events -> error", func(t *testing.T) {
		webhook := &contexts.WebhookContext{
			ID:            "wh-1",
			URL:           "https://hook.example.com/wh-1",
			Configuration: WebhookConfiguration{Events: []string{}},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			Integration: newAuthorizedIntegration(),
			Webhook:     webhook,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one event")
	})

	t.Run("registration error from jira", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"webhookRegistrationResult":[{"errors":["bad jql"]}]}`)),
				},
			},
		}

		webhook := &contexts.WebhookContext{
			ID:  "wh-1",
			URL: "https://hook.example.com/wh-1",
			Configuration: WebhookConfiguration{
				Events:    []string{JiraEventIssueCreated},
				JQLFilter: "bad",
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook:     webhook,
		})

		require.Error(t, err)
	})
}

func Test__JiraWebhookHandler__Cleanup(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("deletes webhooks via jira API", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader(``))},
			},
		}

		webhook := &contexts.WebhookContext{
			ID:       "wh-1",
			Metadata: WebhookMetadata{IDs: []int{42}},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook:     webhook,
		})

		require.NoError(t, err)
		assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	})

	t.Run("no ids -> noop", func(t *testing.T) {
		webhook := &contexts.WebhookContext{
			ID:       "wh-1",
			Metadata: WebhookMetadata{IDs: []int{}},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			Integration: newAuthorizedIntegration(),
			Webhook:     webhook,
		})
		require.NoError(t, err)
	})
}

func Test__JiraWebhookHandler__CompareConfig(t *testing.T) {
	handler := &JiraWebhookHandler{}

	t.Run("identical configurations match", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{JiraEventIssueCreated, JiraEventIssueUpdated}, JQLFilter: "x"}
		b := WebhookConfiguration{Events: []string{JiraEventIssueUpdated, JiraEventIssueCreated}, JQLFilter: "x"}

		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different jql does not match", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{JiraEventIssueCreated}, JQLFilter: "x"}
		b := WebhookConfiguration{Events: []string{JiraEventIssueCreated}, JQLFilter: "y"}

		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different events does not match", func(t *testing.T) {
		a := WebhookConfiguration{Events: []string{JiraEventIssueCreated}}
		b := WebhookConfiguration{Events: []string{JiraEventIssueUpdated}}

		equal, err := handler.CompareConfig(a, b)
		require.NoError(t, err)
		assert.False(t, equal)
	})
}
