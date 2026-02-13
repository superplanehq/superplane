package linear

import (
	"crypto/hmac"
	"crypto/sha256"
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

func Test__OnIssueCreated__HandleWebhook(t *testing.T) {
	trigger := &OnIssueCreated{}

	signatureFor := func(secret string, body []byte) string {
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		return fmt.Sprintf("%x", h.Sum(nil))
	}

	t.Run("missing Linear-Signature -> 403", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Webhook: &contexts.WebhookContext{Secret: "secret"},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing Linear-Signature")
	})

	t.Run("invalid signature -> 403 when secret set", func(t *testing.T) {
		body := []byte(`{"action":"create","type":"Issue","data":{"id":"i1"}}`)
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       map[string][]string{"Linear-Signature": {"wrong"}},
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: "test-secret"},
			Events:        &contexts.EventContext{},
		})
		assert.Equal(t, http.StatusForbidden, code)
		assert.Error(t, err)
	})

	t.Run("action not create -> no emit", func(t *testing.T) {
		body := []byte(`{"action":"update","type":"Issue","data":{"id":"i1"}}`)
		secret := "sec"
		headers := http.Header{}
		headers.Set("Linear-Signature", signatureFor(secret, body))
		eventCtx := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("type not Issue -> no emit", func(t *testing.T) {
		body := []byte(`{"action":"create","type":"Comment","data":{}}`)
		secret := "sec"
		headers := http.Header{}
		headers.Set("Linear-Signature", signatureFor(secret, body))
		eventCtx := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})

	t.Run("valid create -> emit", func(t *testing.T) {
		body := []byte(`{"action":"create","type":"Issue","data":{"id":"i1","teamId":"t1"},"actor":{},"url":"https://linear.app/x","createdAt":"2020-01-01T00:00:00Z","webhookTimestamp":123}`)
		secret := "sec"
		headers := http.Header{}
		headers.Set("Linear-Signature", signatureFor(secret, body))
		eventCtx := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, eventCtx.Count())
		assert.Equal(t, onIssueCreatedPayloadType, eventCtx.Payloads[0].Type)
	})

	t.Run("team filter mismatch -> no emit", func(t *testing.T) {
		body := []byte(`{"action":"create","type":"Issue","data":{"id":"i1","teamId":"other-team"},"actor":{},"url":"","createdAt":"","webhookTimestamp":0}`)
		secret := "sec"
		headers := http.Header{}
		headers.Set("Linear-Signature", signatureFor(secret, body))
		eventCtx := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:          body,
			Headers:       headers,
			Configuration: map[string]any{"team": "my-team"},
			Webhook:       &contexts.WebhookContext{Secret: secret},
			Events:        eventCtx,
		})
		require.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 0, eventCtx.Count())
	})
}

func Test__OnIssueCreated__Setup(t *testing.T) {
	trigger := &OnIssueCreated{}

	t.Run("team not found -> error", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"other","name":"Other","key":"O"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "team-id-1"},
		})
		require.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("requests webhook with team", func(t *testing.T) {
		teamsResp := `{"data":{"teams":{"nodes":[{"id":"team-id-1","name":"Eng","key":"ENG"}]}}}`
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(teamsResp))},
			},
		}
		integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "key"}}
		metaCtx := &contexts.MetadataContext{}
		err := trigger.Setup(core.TriggerContext{
			HTTP:          httpCtx,
			Integration:   integrationCtx,
			Metadata:      metaCtx,
			Configuration: map[string]any{"team": "team-id-1"},
		})
		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		cfg, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "team-id-1", cfg.TeamID)
		assert.False(t, cfg.AllPublicTeams)
		md, _ := metaCtx.Get().(NodeMetadata)
		require.NotNil(t, md.Team)
		assert.Equal(t, "team-id-1", md.Team.ID)
	})

	t.Run("requests webhook without team -> all public teams", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{},
		})
		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
		cfg, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.True(t, cfg.AllPublicTeams)
		assert.Empty(t, cfg.TeamID)
	})
}
