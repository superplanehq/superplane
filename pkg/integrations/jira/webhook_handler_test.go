package jira

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__JiraWebhookHandler__Setup(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			response(http.StatusOK, `{"webhookRegistrationResult":[{"createdWebhookId":12345}]}`),
		},
	}

	appCtx := &contexts.IntegrationContext{
		Metadata: Metadata{CloudID: "cloud-123"},
		CurrentSecrets: map[string]core.IntegrationSecret{
			OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
		},
	}

	webhookCtx := &contexts.WebhookContext{
		URL:           "https://superplane.example/api/v1/webhooks/webhook-id",
		Configuration: WebhookConfiguration{CloudID: "cloud-123"},
	}

	metadata, err := (&JiraWebhookHandler{}).Setup(core.WebhookHandlerContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Webhook:     webhookCtx,
	})

	require.NoError(t, err)
	assert.Equal(t, WebhookMetadata{WebhookID: 12345}, metadata)

	require.Len(t, httpContext.Requests, 1)
	request := httpContext.Requests[0]
	assert.Equal(t, http.MethodPost, request.Method)
	assert.Equal(t, "https://api.atlassian.com/ex/jira/cloud-123/rest/api/3/webhook", request.URL.String())
	assert.Equal(t, "Bearer access-token", request.Header.Get("Authorization"))
}

func Test__JiraWebhookHandler__Cleanup(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			response(http.StatusNoContent, ""),
		},
	}

	appCtx := &contexts.IntegrationContext{
		Metadata: Metadata{CloudID: "cloud-123"},
		CurrentSecrets: map[string]core.IntegrationSecret{
			OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("access-token")},
		},
	}

	err := (&JiraWebhookHandler{}).Cleanup(core.WebhookHandlerContext{
		HTTP:        httpContext,
		Integration: appCtx,
		Webhook: &contexts.WebhookContext{
			Metadata: WebhookMetadata{WebhookID: 12345},
		},
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.atlassian.com/ex/jira/cloud-123/rest/api/3/webhook", httpContext.Requests[0].URL.String())
}

func Test__JiraWebhookHandler__CompareConfig(t *testing.T) {
	matches, err := (&JiraWebhookHandler{}).CompareConfig(
		WebhookConfiguration{CloudID: "cloud-123"},
		WebhookConfiguration{CloudID: "cloud-123"},
	)

	require.NoError(t, err)
	assert.True(t, matches)

	matches, err = (&JiraWebhookHandler{}).CompareConfig(
		WebhookConfiguration{CloudID: "cloud-123"},
		WebhookConfiguration{CloudID: "cloud-456"},
	)

	require.NoError(t, err)
	assert.False(t, matches)
}
