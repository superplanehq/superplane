package linear

import (
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	bugLabelID     = "3f0c1a55-9d2e-4a1b-8c3f-77a1b2c3d4e5"
	factoryLabelID = "8b1d2c3e-4f5a-6b7c-8d9e-0f1a2b3c4d5e"
)

// labelIssueEvent builds an Issue webhook body carrying labelIds, optionally with
// an updatedFrom block describing the previous label set.
func labelIssueEvent(action string, labelIDs []string, updatedFrom map[string]any) map[string]any {
	event := map[string]any{
		"action": action,
		"type":   IssueResourceType,
		"url":    "https://linear.app/acme/issue/ENG-142",
		"data": map[string]any{
			"identifier": "ENG-142",
			"title":      "Deploy pipeline fails on retry",
			"labelIds":   labelIDs,
		},
	}

	if updatedFrom != nil {
		event["updatedFrom"] = updatedFrom
	}

	return event
}

func labelConfig() map[string]any {
	return map[string]any{"team": "t1", "labels": []string{factoryLabelID}}
}

func Test__OnIssueLabel__Setup(t *testing.T) {
	trigger := &OnIssueLabel{}

	t.Run("missing team -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"labels": []string{factoryLabelID}},
		})

		require.ErrorContains(t, err, "team is required")
	})

	t.Run("unknown team -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "other", "labels": []string{factoryLabelID}},
		})

		require.ErrorContains(t, err, "team other not found")
	})

	t.Run("missing labels -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1"},
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	t.Run("empty labels -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "labels": []string{}},
		})

		require.ErrorContains(t, err, "at least one label is required")
	})

	t.Run("requests an issue webhook scoped to the team", func(t *testing.T) {
		integration := integrationWithTeam()
		metadataContext := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			Metadata:      metadataContext,
			Configuration: labelConfig(),
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

func Test__OnIssueLabel__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssueLabel{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: labelConfig(),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	require.ErrorContains(t, err, EventHeader)
}

func Test__OnIssueLabel__HandleWebhook__IgnoresOtherResourceTypes(t *testing.T) {
	trigger := &OnIssueLabel{}
	events := &contexts.EventContext{}

	headers := http.Header{}
	headers.Set(EventHeader, CommentResourceType)

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: labelConfig(),
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueLabel__HandleWebhook__InvalidSignature(t *testing.T) {
	trigger := &OnIssueLabel{}
	events := &contexts.EventContext{}

	ctx := signedRequest(t, labelIssueEvent("create", []string{factoryLabelID}, nil), labelConfig(), events)
	ctx.Headers.Set(SignatureHeader, "deadbeef")

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, "invalid webhook signature")
	assert.Zero(t, events.Count())
}

func Test__OnIssueLabel__HandleWebhook__Emits(t *testing.T) {
	trigger := &OnIssueLabel{}

	t.Run("issue created with the target label", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("create", []string{bugLabelID, factoryLabelID}, nil)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, IssuePayloadType, events.Payloads[0].Type)
	})

	t.Run("target label added on update", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("update",
			[]string{bugLabelID, factoryLabelID},
			map[string]any{"labelIds": []string{bugLabelID}},
		)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnIssueLabel__HandleWebhook__Ignores(t *testing.T) {
	trigger := &OnIssueLabel{}

	t.Run("update that does not touch labels", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("update",
			[]string{bugLabelID, factoryLabelID},
			map[string]any{"title": "Old title"},
		)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("update where the target label was already present", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("update",
			[]string{bugLabelID, factoryLabelID},
			map[string]any{"labelIds": []string{bugLabelID, factoryLabelID}},
		)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("event without the target label", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("create", []string{bugLabelID}, nil)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("issue removed", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := labelIssueEvent("remove", []string{factoryLabelID}, nil)
		ctx := signedRequest(t, body, labelConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}

func Test__OnIssueLabel__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnIssueLabel{}
	example := trigger.ExampleData()

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, IssuePayloadType, example["type"])
	assert.Equal(t, IssueResourceType, data["type"])
}
