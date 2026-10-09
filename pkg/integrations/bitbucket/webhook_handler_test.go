package bitbucket

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
