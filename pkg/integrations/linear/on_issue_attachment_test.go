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

func attachmentEvent(action string) map[string]any {
	return map[string]any{
		"action": action,
		"type":   AttachmentResourceType,
		"data": map[string]any{
			"id":            "a7c41e88-1b2d-4f6a-9c3e-5d8f0a1b2c3d",
			"title":         "Deploy - staging",
			"subtitle":      "Succeeded in 4m 12s",
			"url":           "https://app.superplane.com/runs/8f21c4d0",
			"issueId":       "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9",
			"sourceType":    "unknown",
			"groupBySource": false,
		},
	}
}

func attachmentTriggerConfig() map[string]any {
	return map[string]any{"team": "t1", "actions": []string{"create"}}
}

// signedAttachmentRequest signs the body and sets the Attachment event header,
// since the shared helper defaults to the Issue resource type.
func signedAttachmentRequest(t *testing.T, body, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	t.Helper()

	ctx := signedRequest(t, body, config, events)
	ctx.Headers.Set(EventHeader, AttachmentResourceType)
	return ctx
}

func Test__OnIssueAttachment__Setup(t *testing.T) {
	trigger := &OnIssueAttachment{}

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

	t.Run("empty actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "actions": []string{}},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	t.Run("requests an attachment webhook scoped to the team", func(t *testing.T) {
		integration := integrationWithTeam()
		metadataContext := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			Metadata:      metadataContext,
			Configuration: attachmentTriggerConfig(),
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
		assert.Equal(t, AttachmentResourceType, webhookConfig.ResourceType)
	})
}

func Test__OnIssueAttachment__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssueAttachment{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: attachmentTriggerConfig(),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	require.ErrorContains(t, err, EventHeader)
}

func Test__OnIssueAttachment__HandleWebhook__IgnoresOtherResourceTypes(t *testing.T) {
	trigger := &OnIssueAttachment{}
	events := &contexts.EventContext{}

	headers := http.Header{}
	headers.Set(EventHeader, CommentResourceType)

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: attachmentTriggerConfig(),
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueAttachment__HandleWebhook__InvalidSignature(t *testing.T) {
	trigger := &OnIssueAttachment{}
	events := &contexts.EventContext{}

	ctx := signedAttachmentRequest(t, attachmentEvent("create"), attachmentTriggerConfig(), events)
	ctx.Headers.Set(SignatureHeader, "deadbeef")

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, "invalid webhook signature")
	assert.Zero(t, events.Count())
}

func Test__OnIssueAttachment__HandleWebhook__Success(t *testing.T) {
	trigger := &OnIssueAttachment{}
	events := &contexts.EventContext{}

	ctx := signedAttachmentRequest(t, attachmentEvent("create"), attachmentTriggerConfig(), events)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	require.Equal(t, 1, events.Count())
	assert.Equal(t, AttachmentPayloadType, events.Payloads[0].Type)

	data, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "create", data["action"])
}

func Test__OnIssueAttachment__HandleWebhook__FiltersActions(t *testing.T) {
	trigger := &OnIssueAttachment{}

	t.Run("action not selected is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedAttachmentRequest(t, attachmentEvent("update"), attachmentTriggerConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("empty actions emits nothing", func(t *testing.T) {
		for _, action := range []string{"create", "update", "remove"} {
			events := &contexts.EventContext{}
			ctx := signedAttachmentRequest(t, attachmentEvent(action), map[string]any{"team": "t1", "actions": []string{}}, events)

			code, _, err := trigger.HandleWebhook(ctx)
			assert.Equal(t, http.StatusOK, code)
			require.NoError(t, err)
			assert.Zero(t, events.Count(), "action %q must not be emitted with an empty action list", action)
		}
	})

	t.Run("remove action is delivered when selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedAttachmentRequest(t, attachmentEvent("remove"), map[string]any{"team": "t1", "actions": []string{"create", "remove"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnIssueAttachment__HandleWebhook__EnrichesWithIssue(t *testing.T) {
	trigger := &OnIssueAttachment{}

	t.Run("adds the issue Linear omits from the payload", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedAttachmentRequest(t, attachmentEvent("create"), attachmentTriggerConfig(), events)
		ctx.HTTP = &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"data":{"issue":{"id":"2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9","identifier":"ENG-142","title":"Deploy pipeline fails on retry","url":"https://linear.app/acme/issue/ENG-142"}}}`),
			},
		}
		ctx.Integration = newAuthorizedIntegration()

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())

		data, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		attachment, ok := data["data"].(map[string]any)
		require.True(t, ok)

		issue, ok := attachment["issue"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ENG-142", issue["identifier"])
		assert.Equal(t, "Deploy pipeline fails on retry", issue["title"])
		assert.Equal(t, "https://linear.app/acme/issue/ENG-142", issue["url"])

		// issueId stays put, so the enrichment is purely additive.
		assert.Equal(t, "2174add1-f7c8-44e3-bbf3-2d60b5ea8bc9", attachment["issueId"])
	})

	t.Run("a failed lookup still delivers the event", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedAttachmentRequest(t, attachmentEvent("create"), attachmentTriggerConfig(), events)
		ctx.HTTP = &contexts.HTTPContext{
			Responses: []*http.Response{
				jsonResponse(`{"errors":[{"message":"Entity not found"}]}`),
			},
		}
		ctx.Integration = newAuthorizedIntegration()

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())

		data, ok := events.Payloads[0].Data.(map[string]any)
		require.True(t, ok)
		attachment, ok := data["data"].(map[string]any)
		require.True(t, ok)
		assert.NotContains(t, attachment, "issue")
	})

	t.Run("no HTTP context skips enrichment without failing", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedAttachmentRequest(t, attachmentEvent("create"), attachmentTriggerConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		require.Equal(t, 1, events.Count())
	})
}

func Test__OnIssueAttachment__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnIssueAttachment{}
	example := trigger.ExampleData()

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, AttachmentPayloadType, example["type"])
	assert.Equal(t, "create", data["action"])
	assert.Equal(t, AttachmentResourceType, data["type"])
}
