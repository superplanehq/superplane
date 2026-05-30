package railway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

type integrationWebhookContext struct {
	id            string
	url           string
	configuration any
	metadata      any
	secret        []byte
}

func (w *integrationWebhookContext) GetID() string              { return w.id }
func (w *integrationWebhookContext) GetURL() string             { return w.url }
func (w *integrationWebhookContext) GetSecret() ([]byte, error) { return w.secret, nil }
func (w *integrationWebhookContext) GetMetadata() any           { return w.metadata }
func (w *integrationWebhookContext) GetConfiguration() any      { return w.configuration }
func (w *integrationWebhookContext) SetSecret(secret []byte) error {
	w.secret = secret
	return nil
}

func Test__Railway__WebhookHandler__CompareConfig(t *testing.T) {
	h := &RailwayWebhookHandler{}

	equal, err := h.CompareConfig(
		WebhookConfiguration{ProjectID: "p-1"},
		WebhookConfiguration{ProjectID: "p-1"},
	)
	require.NoError(t, err)
	assert.True(t, equal)

	equal, err = h.CompareConfig(
		WebhookConfiguration{ProjectID: "p-1"},
		WebhookConfiguration{ProjectID: "p-2"},
	)
	require.NoError(t, err)
	assert.False(t, equal)
}

func Test__Railway__WebhookHandler__Merge(t *testing.T) {
	h := &RailwayWebhookHandler{}

	merged, changed, err := h.Merge(
		WebhookConfiguration{ProjectID: "p-1", EventTypes: []string{"Deployment.deployed"}},
		WebhookConfiguration{ProjectID: "p-1", EventTypes: []string{"Deployment.failed"}},
	)
	require.NoError(t, err)
	assert.True(t, changed)

	expected := WebhookConfiguration{
		ProjectID:  "p-1",
		EventTypes: []string{"Deployment.deployed", "Deployment.failed"},
	}
	assert.Equal(t, expected, merged)
}

func Test__Railway__WebhookHandler__Setup(t *testing.T) {
	h := &RailwayWebhookHandler{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"project":{"id":"p-1","name":"Project 1","workspaceId":"w-2"}}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"notificationRuleCreate":{"id":"rule-123"}}}`)),
			},
		},
	}

	webhookCtx := &integrationWebhookContext{
		id:            "wh-123",
		url:           "https://hook.superplane.dev",
		configuration: WebhookConfiguration{ProjectID: "p-1", EventTypes: []string{"Deployment.deployed"}},
	}

	intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
	_ = intCtx.SetSecret("apiToken", []byte("test-token"))

	metadata, err := h.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Webhook:     webhookCtx,
		Integration: intCtx,
	})
	require.NoError(t, err)

	storedMetadata, ok := metadata.(WebhookMetadata)
	require.True(t, ok)
	assert.Equal(t, "rule-123", storedMetadata.RuleID)
	assert.Equal(t, "w-2", storedMetadata.WorkspaceID)
	require.Len(t, httpCtx.Requests, 2)
}

func Test__Railway__WebhookHandler__Setup_missingWorkspaceId(t *testing.T) {
	h := &RailwayWebhookHandler{}

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"data":{"project":{"id":"p-1","name":"Project 1"}}}`)),
			},
		},
	}

	intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
	_ = intCtx.SetSecret("apiToken", []byte("test-token"))

	_, err := h.Setup(core.WebhookHandlerContext{
		HTTP:        httpCtx,
		Webhook:     &integrationWebhookContext{configuration: WebhookConfiguration{ProjectID: "p-1"}},
		Integration: intCtx,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workspaceId")
}

func Test__Railway__WebhookHandler__Cleanup(t *testing.T) {
	h := &RailwayWebhookHandler{}
	logger := logger.DiscardLogger()

	t.Run("success", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"notificationRuleDelete":true}}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
		_ = intCtx.SetSecret("apiToken", []byte("test-token"))

		err := h.Cleanup(core.WebhookHandlerContext{
			HTTP:   httpCtx,
			Logger: logger,
			Webhook: &integrationWebhookContext{
				metadata: WebhookMetadata{RuleID: "rule-123", WorkspaceID: "w-1"},
			},
			Integration: intCtx,
		})
		require.NoError(t, err)
	})

	t.Run("ignores not authorized error", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errors":[{"message":"Not Authorized"}]}`)),
				},
			},
		}

		intCtx := &contexts.IntegrationContext{NewSetupFlow: true}
		_ = intCtx.SetSecret("apiToken", []byte("test-token"))

		err := h.Cleanup(core.WebhookHandlerContext{
			HTTP:   httpCtx,
			Logger: logger,
			Webhook: &integrationWebhookContext{
				metadata: WebhookMetadata{RuleID: "rule-123", WorkspaceID: "w-1"},
			},
			Integration: intCtx,
		})
		require.NoError(t, err) // Should succeed because Not Authorized is caught and handled
	})
}
