package linear

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const testWebhookSecret = "webhook-secret"

// signedRequest builds a webhook request signed the way Linear signs one.
func signedRequest(t *testing.T, body map[string]any, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	t.Helper()

	raw, err := json.Marshal(body)
	require.NoError(t, err)

	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(raw)

	headers := http.Header{}
	headers.Set(EventHeader, IssueResourceType)
	headers.Set(SignatureHeader, hex.EncodeToString(mac.Sum(nil)))

	return core.WebhookRequestContext{
		Headers:       headers,
		Body:          raw,
		Configuration: config,
		Webhook:       &contexts.NodeWebhookContext{Secret: testWebhookSecret},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	}
}

func issueEvent(action string, labels []map[string]any) map[string]any {
	return map[string]any{
		"action": action,
		"type":   IssueResourceType,
		"url":    "https://linear.app/acme/issue/ENG-142",
		"data": map[string]any{
			"identifier": "ENG-142",
			"title":      "Deploy pipeline fails on retry",
			"labels":     labels,
		},
	}
}

func Test__OnIssue__Setup(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"actions": []string{"create"}},
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "other", "actions": []string{"create"}},
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("missing actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1"},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	// The shared multi-select validation lets an empty list satisfy Required,
	// so Setup has to reject it or the trigger would never match anything.
	t.Run("empty actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "actions": []string{}},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	t.Run("requests a webhook scoped to the team", func(t *testing.T) {
		integration := integrationWithTeam()
		metadataContext := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			Metadata:      metadataContext,
			Configuration: map[string]any{"team": "t1", "actions": []string{"create"}},
		})

		require.NoError(t, err)

		metadata, ok := metadataContext.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, metadata.Team)
		assert.Equal(t, "ENG", metadata.Team.Key)

		require.Len(t, integration.WebhookRequests, 1)
		webhookConfig, ok := integration.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, "t1", webhookConfig.TeamID)
		assert.Equal(t, IssueResourceType, webhookConfig.ResourceType)
	})
}

func Test__OnIssue__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssue{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"team": "t1", "actions": []string{"create"}},
	})

	assert.Equal(t, http.StatusBadRequest, code)
	require.ErrorContains(t, err, EventHeader)
}

func Test__OnIssue__HandleWebhook__IgnoresOtherResourceTypes(t *testing.T) {
	trigger := &OnIssue{}
	events := &contexts.EventContext{}

	headers := http.Header{}
	headers.Set(EventHeader, "Comment")

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"team": "t1", "actions": []string{"create"}},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssue__HandleWebhook__InvalidSignature(t *testing.T) {
	trigger := &OnIssue{}
	events := &contexts.EventContext{}

	ctx := signedRequest(t, issueEvent("create", nil), map[string]any{"team": "t1", "actions": []string{"create"}}, events)
	ctx.Headers.Set(SignatureHeader, "deadbeef")

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, "invalid webhook signature")
	assert.Zero(t, events.Count())
}

func Test__OnIssue__HandleWebhook__MissingSignature(t *testing.T) {
	trigger := &OnIssue{}
	events := &contexts.EventContext{}

	ctx := signedRequest(t, issueEvent("create", nil), map[string]any{"team": "t1", "actions": []string{"create"}}, events)
	ctx.Headers.Del(SignatureHeader)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, SignatureHeader)
}

func Test__OnIssue__HandleWebhook__Success(t *testing.T) {
	trigger := &OnIssue{}
	events := &contexts.EventContext{}

	ctx := signedRequest(t, issueEvent("create", nil), map[string]any{"team": "t1", "actions": []string{"create"}}, events)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	require.Equal(t, 1, events.Count())
	assert.Equal(t, IssuePayloadType, events.Payloads[0].Type)

	data, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "create", data["action"])
}

func Test__OnIssue__HandleWebhook__FiltersActions(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("action not selected is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedRequest(t, issueEvent("update", nil), map[string]any{"team": "t1", "actions": []string{"create"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	// Regression: an empty list must match nothing rather than falling through
	// and emitting every action.
	t.Run("empty actions emits nothing", func(t *testing.T) {
		for _, action := range []string{"create", "update", "remove"} {
			events := &contexts.EventContext{}
			ctx := signedRequest(t, issueEvent(action, nil), map[string]any{"team": "t1", "actions": []string{}}, events)

			code, _, err := trigger.HandleWebhook(ctx)
			assert.Equal(t, http.StatusOK, code)
			require.NoError(t, err)
			assert.Zero(t, events.Count(), "action %q must not be emitted with an empty action list", action)
		}
	})

	t.Run("missing actions key emits nothing", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedRequest(t, issueEvent("create", nil), map[string]any{"team": "t1"}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("remove action is delivered when selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedRequest(t, issueEvent("remove", nil), map[string]any{"team": "t1", "actions": []string{"create", "remove"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnIssue__HandleWebhook__FiltersLabels(t *testing.T) {
	trigger := &OnIssue{}
	labelPredicates := []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "backend"}}

	t.Run("label match", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := issueEvent("create", []map[string]any{{"name": "bug"}, {"name": "backend"}})
		ctx := signedRequest(t, body, map[string]any{"team": "t1", "actions": []string{"create"}, "labels": labelPredicates}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("label no match", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := issueEvent("create", []map[string]any{{"name": "bug"}})
		ctx := signedRequest(t, body, map[string]any{"team": "t1", "actions": []string{"create"}, "labels": labelPredicates}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("issue without labels does not match", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := issueEvent("create", nil)
		ctx := signedRequest(t, body, map[string]any{"team": "t1", "actions": []string{"create"}, "labels": labelPredicates}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}

func Test__OnIssue__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnIssue{}
	example := trigger.ExampleData()

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, IssuePayloadType, example["type"])
	assert.Equal(t, "create", data["action"])
	assert.Equal(t, IssueResourceType, data["type"])
}
