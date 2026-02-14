package linear

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("no-op returns 200", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{})
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})
}

func Test__OnIssue__OnIntegrationMessage(t *testing.T) {
	trigger := &OnIssue{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("type not Issue -> no emit", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "create",
				"type":   "Comment",
				"data":   map[string]any{},
			},
			Configuration: map[string]any{},
			Events:        eventCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("create Issue -> emit", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action":           "create",
				"type":             "Issue",
				"data":             map[string]any{"id": "i1", "teamId": "t1"},
				"actor":            map[string]any{"name": "Bob"},
				"url":              "https://linear.app/x",
				"createdAt":        "2020-01-01T00:00:00Z",
				"webhookTimestamp": float64(123),
			},
			Configuration: map[string]any{},
			Events:        eventCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, onIssuePayloadType, eventCtx.Payloads[0].Type)
	})

	t.Run("update Issue -> emit", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "update",
				"type":   "Issue",
				"data":   map[string]any{"id": "i1", "teamId": "t1"},
			},
			Configuration: map[string]any{},
			Events:        eventCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, onIssuePayloadType, eventCtx.Payloads[0].Type)
	})

	t.Run("team filter mismatch -> no emit", func(t *testing.T) {
		eventCtx := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"action": "create",
				"type":   "Issue",
				"data":   map[string]any{"id": "i1", "teamId": "other-team"},
			},
			Configuration: map[string]any{"team": "my-team"},
			Events:        eventCtx,
			Logger:        logger,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})
}

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("team not found -> error", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"other","name":"Other","key":"O"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("key")},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "team-id-1"},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("subscribes with team", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"team-id-1","name":"Eng","key":"ENG"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Secrets: map[string]core.IntegrationSecret{
				OAuthAccessToken: {Name: OAuthAccessToken, Value: []byte("key")},
			},
		}
		metaCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      metaCtx,
			Configuration: map[string]any{"team": "team-id-1"},
		})
		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		md, ok := metaCtx.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, md.Team)
		assert.Equal(t, "team-id-1", md.Team.ID)
		require.NotNil(t, md.SubscriptionID)
	})

	t.Run("subscribes without team", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		metaCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metaCtx,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		require.Len(t, integrationCtx.Subscriptions, 1)
		md, ok := metaCtx.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, md.SubscriptionID)
	})

	t.Run("re-setup reuses existing subscription", func(t *testing.T) {
		existingSubID := "existing-sub-id"
		integrationCtx := &contexts.IntegrationContext{}
		metaCtx := &contexts.MetadataContext{
			Metadata: NodeMetadata{
				SubscriptionID: &existingSubID,
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      metaCtx,
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		assert.Empty(t, integrationCtx.Subscriptions)
		md, ok := metaCtx.Get().(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, md.SubscriptionID)
		assert.Equal(t, "existing-sub-id", *md.SubscriptionID)
	})
}
