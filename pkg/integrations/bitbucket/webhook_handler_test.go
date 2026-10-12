package bitbucket

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/bitbucketapp"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestBitbucketWebhookHandlerForgeUsesLocalSubscription(t *testing.T) {
	bitbucketapp.SetSystemTokenSource(func(string) (string, time.Time, error) { return "token", time.Now().Add(time.Hour), nil })
	t.Cleanup(func() { bitbucketapp.SetSystemTokenSource(nil) })
	integration := &contexts.IntegrationContext{Metadata: map[string]any{
		"authType": AuthTypeForgeApp, "forgeInstallationId": "installation-1", "workspace": map[string]any{"slug": "acme"},
	}}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{{StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{"uuid":"{repo-1}","full_name":"acme/widgets"}`)),
	}}}
	webhook := &contexts.WebhookContext{Configuration: WebhookConfiguration{
		RepositorySlug: "widgets", EventTypes: []string{"pullrequest:fulfilled", "pullrequest:rejected"},
	}}
	ctx := core.WebhookHandlerContext{HTTP: httpContext, Integration: integration, Webhook: webhook}
	handler := &BitbucketWebhookHandler{}
	metadata, err := handler.Setup(ctx)
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, &BitbucketWebhook{RepositoryUUID: "{repo-1}", RepositoryFullName: "acme/widgets"}, metadata)
	require.NoError(t, handler.Cleanup(ctx))
	webhook.Configuration = WebhookConfiguration{RepositorySlug: "widgets", EventTypes: []string{"repo:push"}}
	_, err = handler.Setup(ctx)
	require.ErrorContains(t, err, "does not support event")
	assert.Len(t, httpContext.Requests, 1)

	// Build-status subscriptions share the Forge bridge with PR events
	httpContext.Responses = []*http.Response{{StatusCode: http.StatusOK,
		Body: io.NopCloser(strings.NewReader(`{"uuid":"{repo-1}","full_name":"acme/widgets"}`)),
	}}
	webhook.Configuration = WebhookConfiguration{
		RepositorySlug: "widgets",
		EventTypes:     []string{"pullrequest:created", "repo:commit_status_created", "repo:commit_status_updated", "pullrequest:comment_created"},
	}
	metadata, err = handler.Setup(ctx)
	require.NoError(t, err)
	assert.Equal(t, &BitbucketWebhook{RepositoryUUID: "{repo-1}", RepositoryFullName: "acme/widgets"}, metadata)
}

func Test__BitbucketWebhookHandler__SetupUpdatesExistingHook(t *testing.T) {
	handler := &BitbucketWebhookHandler{}
	integration := &contexts.IntegrationContext{
		Configuration: map[string]any{"token": "token"},
		Metadata: map[string]any{
			"authType":  AuthTypeWorkspaceAccessToken,
			"workspace": map[string]any{"slug": "acme"},
		},
	}
	config := WebhookConfiguration{
		EventTypes:     []string{"pullrequest:created", "pullrequest:comment_created"},
		RepositorySlug: "widgets",
	}

	t.Run("updates the stored hook instead of creating another", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"uuid":"{hook-1}","active":true}`))},
		}}
		webhook := &contexts.WebhookContext{
			URL:           "https://example.com/hook",
			Secret:        []byte("secret"),
			Configuration: config,
			Metadata:      map[string]any{"uuid": "{hook-1}"},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     webhook,
		})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Contains(t, httpContext.Requests[0].URL.Path, "{hook-1}")
		stored, ok := metadata.(*BitbucketWebhook)
		require.True(t, ok)
		assert.Equal(t, "{hook-1}", stored.UUID)
	})

	t.Run("creates a hook when the stored hook is gone", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"not found"}}`))},
			{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(`{"uuid":"{hook-2}","active":true}`))},
		}}
		webhook := &contexts.WebhookContext{
			URL:           "https://example.com/hook",
			Secret:        []byte("secret"),
			Configuration: config,
			Metadata:      map[string]any{"uuid": "{hook-1}"},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: integration,
			Webhook:     webhook,
		})
		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, http.MethodPut, httpContext.Requests[0].Method)
		assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
		stored, ok := metadata.(*BitbucketWebhook)
		require.True(t, ok)
		assert.Equal(t, "{hook-2}", stored.UUID)
	})
}
