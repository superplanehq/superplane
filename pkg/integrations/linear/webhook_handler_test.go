package linear

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__WebhookHandler__CompareConfig(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("same team and resource type match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
		)

		require.NoError(t, err)
		assert.True(t, equal)
	})

	t.Run("different teams do not match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			WebhookConfiguration{TeamID: "t2", ResourceType: IssueResourceType},
		)

		require.NoError(t, err)
		assert.False(t, equal)
	})

	t.Run("different resource types do not match", func(t *testing.T) {
		equal, err := handler.CompareConfig(
			WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			WebhookConfiguration{TeamID: "t1", ResourceType: "Comment"},
		)

		require.NoError(t, err)
		assert.False(t, equal)
	})
}

func Test__WebhookHandler__Setup(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("creates the webhook with SuperPlane's secret", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"webhookCreate":{"success":true,"webhook":{"id":"w1","url":"https://sp.test/hook"}}}}`),
			},
		}

		metadata, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte(testWebhookSecret),
				Configuration: WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			},
		})

		require.NoError(t, err)
		webhookMetadata, ok := metadata.(*WebhookMetadata)
		require.True(t, ok)
		assert.Equal(t, "w1", webhookMetadata.ID)

		input := webhookInputFromRequest(t, httpContext)
		assert.Equal(t, "https://sp.test/hook", input["url"])
		assert.Equal(t, testWebhookSecret, input["secret"])
		assert.Equal(t, "t1", input["teamId"])
	})

	t.Run("permission failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Only admins can manage webhooks"}]}`),
			},
		}

		_, err := handler.Setup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				URL:           "https://sp.test/hook",
				Secret:        []byte(testWebhookSecret),
				Configuration: WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			},
		})

		require.ErrorContains(t, err, "Only admins can manage webhooks")
	})
}

func Test__WebhookHandler__Cleanup(t *testing.T) {
	handler := &LinearWebhookHandler{}

	t.Run("deletes the webhook", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"webhookDelete":{"success":true}}}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "w1"},
				Configuration: WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			},
		})

		require.NoError(t, err)
	})

	t.Run("delete failure is surfaced", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Entity not found"}]}`),
			},
		}

		err := handler.Cleanup(core.WebhookHandlerContext{
			HTTP:        httpContext,
			Integration: newAuthorizedIntegration(),
			Webhook: &contexts.WebhookContext{
				Metadata:      WebhookMetadata{ID: "w1"},
				Configuration: WebhookConfiguration{TeamID: "t1", ResourceType: IssueResourceType},
			},
		})

		require.ErrorContains(t, err, "Entity not found")
	})
}
