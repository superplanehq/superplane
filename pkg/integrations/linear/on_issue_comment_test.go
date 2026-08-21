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

func commentEvent(action string) map[string]any {
	return map[string]any{
		"action": action,
		"type":   CommentResourceType,
		"url":    "https://linear.app/acme/issue/ENG-142#comment-c1",
		"data": map[string]any{
			"id":   "d3f9b0a2-4c1e-4b6a-9f2e-8a1b2c3d4e5f",
			"body": "Deploy pipeline is green again.",
			"issue": map[string]any{
				"identifier": "ENG-142",
				"title":      "Deploy pipeline fails on retry",
			},
			"user": map[string]any{"name": "Ada Lovelace"},
		},
	}
}

func commentEventWithBody(action, body string) map[string]any {
	event := commentEvent(action)
	event["data"].(map[string]any)["body"] = body
	return event
}

func commentConfig() map[string]any {
	return map[string]any{"team": "t1", "actions": []string{"create"}}
}

func commentConfigWithFilter(filter string) map[string]any {
	return map[string]any{"team": "t1", "actions": []string{"create"}, "contentFilter": filter}
}

// signedCommentRequest signs the body and sets the Comment event header, since
// the shared helper defaults to the Issue resource type.
func signedCommentRequest(t *testing.T, body, config map[string]any, events *contexts.EventContext) core.WebhookRequestContext {
	t.Helper()

	ctx := signedRequest(t, body, config, events)
	ctx.Headers.Set(EventHeader, CommentResourceType)
	return ctx
}

func Test__OnIssueComment__Setup(t *testing.T) {
	trigger := &OnIssueComment{}

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

	t.Run("empty actions -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"team": "t1", "actions": []string{}},
		})

		require.ErrorContains(t, err, "at least one action is required")
	})

	t.Run("invalid content filter pattern -> error", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: commentConfigWithFilter("[invalid(regex"),
		})

		require.ErrorContains(t, err, "invalid content filter pattern")
	})

	t.Run("valid content filter is accepted", func(t *testing.T) {
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationWithTeam(),
			Metadata:      &contexts.MetadataContext{},
			Configuration: commentConfigWithFilter("^/deploy"),
		})

		require.NoError(t, err)
	})

	t.Run("requests a comment webhook scoped to the team", func(t *testing.T) {
		integration := integrationWithTeam()
		metadataContext := &contexts.MetadataContext{}

		err := trigger.Setup(core.TriggerContext{
			Integration:   integration,
			Metadata:      metadataContext,
			Configuration: commentConfig(),
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
		assert.Equal(t, CommentResourceType, webhookConfig.ResourceType)
	})
}

func Test__OnIssueComment__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssueComment{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: commentConfig(),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	require.ErrorContains(t, err, EventHeader)
}

func Test__OnIssueComment__HandleWebhook__IgnoresOtherResourceTypes(t *testing.T) {
	trigger := &OnIssueComment{}
	events := &contexts.EventContext{}

	headers := http.Header{}
	headers.Set(EventHeader, IssueResourceType)

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: commentConfig(),
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__InvalidSignature(t *testing.T) {
	trigger := &OnIssueComment{}
	events := &contexts.EventContext{}

	ctx := signedCommentRequest(t, commentEvent("create"), commentConfig(), events)
	ctx.Headers.Set(SignatureHeader, "deadbeef")

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	require.ErrorContains(t, err, "invalid webhook signature")
	assert.Zero(t, events.Count())
}

func Test__OnIssueComment__HandleWebhook__Success(t *testing.T) {
	trigger := &OnIssueComment{}
	events := &contexts.EventContext{}

	ctx := signedCommentRequest(t, commentEvent("create"), commentConfig(), events)

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	require.NoError(t, err)

	require.Equal(t, 1, events.Count())
	assert.Equal(t, CommentPayloadType, events.Payloads[0].Type)

	data, ok := events.Payloads[0].Data.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "create", data["action"])
}

func Test__OnIssueComment__HandleWebhook__FiltersActions(t *testing.T) {
	trigger := &OnIssueComment{}

	t.Run("action not selected is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedCommentRequest(t, commentEvent("update"), commentConfig(), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("empty actions emits nothing", func(t *testing.T) {
		for _, action := range []string{"create", "update", "remove"} {
			events := &contexts.EventContext{}
			ctx := signedCommentRequest(t, commentEvent(action), map[string]any{"team": "t1", "actions": []string{}}, events)

			code, _, err := trigger.HandleWebhook(ctx)
			assert.Equal(t, http.StatusOK, code)
			require.NoError(t, err)
			assert.Zero(t, events.Count(), "action %q must not be emitted with an empty action list", action)
		}
	})

	t.Run("remove action is delivered when selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedCommentRequest(t, commentEvent("remove"), map[string]any{"team": "t1", "actions": []string{"create", "remove"}}, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})
}

func Test__OnIssueComment__HandleWebhook__FiltersContent(t *testing.T) {
	trigger := &OnIssueComment{}

	t.Run("matching body is emitted", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := commentEventWithBody("create", "/deploy staging")
		ctx := signedCommentRequest(t, body, commentConfigWithFilter("^/deploy"), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("non-matching body is ignored", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedCommentRequest(t, commentEvent("create"), commentConfigWithFilter("^/deploy"), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("pattern matches anywhere in the body", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := commentEventWithBody("create", "please /deploy staging now")
		ctx := signedCommentRequest(t, body, commentConfigWithFilter("/deploy"), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("empty filter emits every comment", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedCommentRequest(t, commentEvent("create"), commentConfigWithFilter(""), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("comment without a body does not match", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := commentEvent("remove")
		delete(body["data"].(map[string]any), "body")
		config := map[string]any{"team": "t1", "actions": []string{"remove"}, "contentFilter": "^/deploy"}
		ctx := signedCommentRequest(t, body, config, events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})

	t.Run("invalid pattern -> bad request", func(t *testing.T) {
		events := &contexts.EventContext{}
		ctx := signedCommentRequest(t, commentEvent("create"), commentConfigWithFilter("[invalid(regex"), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusBadRequest, code)
		require.ErrorContains(t, err, "invalid content filter pattern")
		assert.Zero(t, events.Count())
	})

	t.Run("filter is not reached for unselected actions", func(t *testing.T) {
		events := &contexts.EventContext{}
		body := commentEventWithBody("update", "/deploy staging")
		ctx := signedCommentRequest(t, body, commentConfigWithFilter("^/deploy"), events)

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		require.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}

func Test__OnIssueComment__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnIssueComment{}
	example := trigger.ExampleData()

	data, ok := example["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, CommentPayloadType, example["type"])
	assert.Equal(t, "create", data["action"])
	assert.Equal(t, CommentResourceType, data["type"])
}
