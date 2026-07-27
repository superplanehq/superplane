package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnMergeRequest__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnMergeRequest{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusBadRequest, code)
	assert.ErrorContains(t, err, "X-Gitlab-Event")
}

func Test__OnMergeRequest__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnMergeRequest{}
	events := &contexts.EventContext{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Issue Hook", "token"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__OnMergeRequest__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnMergeRequest{}

	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Merge Request Hook", "wrong"),
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusForbidden, code)
	assert.ErrorContains(t, err, "invalid webhook token")
}

func Test__OnMergeRequest__HandleWebhook__ActionMatch(t *testing.T) {
	trigger := &OnMergeRequest{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"action": "open",
			"title":  "New MR",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers: gitlabHeaders("Merge Request Hook", "token"),
		Body:    body,
		Configuration: map[string]any{
			"project": "123",
			"actions": []string{"open"},
		},
		Webhook: &contexts.NodeWebhookContext{Secret: "token"},
		Events:  events,
		Logger:  log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Equal(t, 1, events.Count())
	assert.Equal(t, "gitlab.mergeRequest", events.Payloads[0].Type)
}

func Test__OnMergeRequest__HandleWebhook__ActionMismatch(t *testing.T) {
	trigger := &OnMergeRequest{}

	body, _ := json.Marshal(map[string]any{
		"object_attributes": map[string]any{
			"action": "merge",
		},
	})

	events := &contexts.EventContext{}
	code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
		Headers:       gitlabHeaders("Merge Request Hook", "token"),
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       &contexts.NodeWebhookContext{Secret: "token"},
		Events:        events,
		Logger:        log.NewEntry(log.New()),
	})

	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, events.Count())
}

func Test__MergeRequestDerivedActions(t *testing.T) {
	t.Run("no changes and no oldrev", func(t *testing.T) {
		assert.Empty(t, mergeRequestDerivedActions(map[string]any{}, nil))
	})

	t.Run("synchronize from oldrev", func(t *testing.T) {
		attrs := map[string]any{"oldrev": "e59094b8de0f2f91abbe4760a52d9137260252d8"}
		assert.Equal(t, []string{"synchronize"}, mergeRequestDerivedActions(attrs, nil))
	})

	t.Run("labeled", func(t *testing.T) {
		changes := map[string]any{
			"labels": map[string]any{
				"previous": []any{},
				"current":  []any{map[string]any{"id": float64(1), "title": "bug"}},
			},
		}
		assert.Equal(t, []string{"labeled"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("review requested and removed", func(t *testing.T) {
		changes := map[string]any{
			"reviewers": []any{
				[]any{map[string]any{"id": float64(1)}},
				[]any{map[string]any{"id": float64(2)}},
			},
		}
		assert.ElementsMatch(t, []string{"review_requested", "review_request_removed"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("ready for review", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "Draft: add feature", "current": "add feature"},
		}
		assert.Equal(t, []string{"ready_for_review"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("converted to draft", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "add feature", "current": "Draft: add feature"},
		}
		assert.Equal(t, []string{"converted_to_draft"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("draft toggle does not also derive edited", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "add feature", "current": "[Draft] add feature"},
		}
		assert.Equal(t, []string{"converted_to_draft"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("draft toggle combined with a real title edit derives both", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "(Draft) add feature", "current": "add better feature"},
		}
		assert.ElementsMatch(t, []string{"ready_for_review", "edited"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("auto merge enabled and disabled", func(t *testing.T) {
		enabled := map[string]any{
			"merge_when_pipeline_succeeds": map[string]any{"previous": false, "current": true},
		}
		assert.Equal(t, []string{"auto_merge_enabled"}, mergeRequestDerivedActions(map[string]any{}, enabled))

		disabled := map[string]any{
			"merge_when_pipeline_succeeds": map[string]any{"previous": true, "current": false},
		}
		assert.Equal(t, []string{"auto_merge_disabled"}, mergeRequestDerivedActions(map[string]any{}, disabled))
	})

	t.Run("milestoned and demilestoned", func(t *testing.T) {
		milestoned := map[string]any{
			"milestone_id": map[string]any{"previous": nil, "current": float64(3)},
		}
		assert.Equal(t, []string{"milestoned"}, mergeRequestDerivedActions(map[string]any{}, milestoned))

		demilestoned := map[string]any{
			"milestone_id": map[string]any{"previous": float64(3), "current": nil},
		}
		assert.Equal(t, []string{"demilestoned"}, mergeRequestDerivedActions(map[string]any{}, demilestoned))

		swapped := map[string]any{
			"milestone_id": map[string]any{"previous": float64(3), "current": float64(4)},
		}
		assert.Equal(t, []string{"milestoned"}, mergeRequestDerivedActions(map[string]any{}, swapped))
	})

	t.Run("edited", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "old", "current": "new"},
		}
		assert.Equal(t, []string{"edited"}, mergeRequestDerivedActions(map[string]any{}, changes))
	})

	t.Run("no-op title entry yields no derived actions", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "same", "current": "same"},
		}
		assert.Empty(t, mergeRequestDerivedActions(map[string]any{}, changes))
	})
}

func Test__IsDraftTitle(t *testing.T) {
	assert.True(t, isDraftTitle("Draft: add feature"))
	assert.True(t, isDraftTitle("[Draft] add feature"))
	assert.True(t, isDraftTitle("(Draft) add feature"))
	assert.True(t, isDraftTitle("draft: lowercase"))
	assert.False(t, isDraftTitle("add feature"))
	assert.False(t, isDraftTitle("WIP: add feature"))
}

func Test__DraftlessTitle(t *testing.T) {
	assert.Equal(t, "add feature", draftlessTitle("Draft: add feature"))
	assert.Equal(t, "add feature", draftlessTitle("[Draft] add feature"))
	assert.Equal(t, "add feature", draftlessTitle("add feature"))
}

func Test__OnMergeRequest__HandleWebhook__DerivedActions(t *testing.T) {
	trigger := &OnMergeRequest{}
	headers := gitlabHeaders("Merge Request Hook", "token")
	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}

	t.Run("ready for review matches without update selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		body, _ := json.Marshal(map[string]any{
			"object_attributes": map[string]any{"action": "update"},
			"changes": map[string]any{
				"title": map[string]any{"previous": "Draft: add feature", "current": "add feature"},
			},
		})

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"ready_for_review"}},
			Webhook:       webhookCtx,
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("no match when derived action is not selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		body, _ := json.Marshal(map[string]any{
			"object_attributes": map[string]any{"action": "update"},
			"changes": map[string]any{
				"title": map[string]any{"previous": "Draft: add feature", "current": "add feature"},
			},
		})

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"labeled"}},
			Webhook:       webhookCtx,
			Events:        events,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, events.Count())
	})
}
