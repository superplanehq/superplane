package gitlab

import (
	"encoding/json"
	"net/http"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnIssue__HandleWebhook__MissingEventHeader(t *testing.T) {
	trigger := &OnIssue{}

	ctx := core.WebhookRequestContext{
		Headers:       http.Header{},
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Gitlab-Event")
}

func Test__OnIssue__HandleWebhook__WrongEventType(t *testing.T) {
	trigger := &OnIssue{}

	eventsCtx := &contexts.EventContext{}
	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Push Hook")

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)
	assert.Zero(t, eventsCtx.Count())
}

func Test__OnIssue__HandleWebhook__InvalidToken(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "wrong-token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "correct-token"}

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          []byte(`{}`),
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       webhookCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusForbidden, code)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook token")
}

func Test__OnIssue__HandleWebhook__StateNotOpened(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "closed",
			"action": "close",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"close"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 1, eventsCtx.Count())
	assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook__Success(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "opened",
			"action": "open",
			"title":  "Test Issue",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"open"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 1, eventsCtx.Count())
	assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
}

func Test__OnIssue__HandleWebhook__Filters(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}

	baseAttributes := map[string]any{
		"state":  "opened",
		"action": "open",
	}

	t.Run("label match", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": baseAttributes,
			"labels": []map[string]any{
				{"title": "bug"},
				{"title": "backend"},
			},
		}
		body, _ := json.Marshal(data)

		ctx := core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}, "labels": []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "backend"}}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)

		assert.Equal(t, 1, eventsCtx.Count())
		assert.Equal(t, "gitlab.issue", eventsCtx.Payloads[0].Type)
	})

	t.Run("label no match", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": baseAttributes,
			"labels": []map[string]any{
				{"title": "bug"},
			},
		}
		body, _ := json.Marshal(data)

		ctx := core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"open"}, "labels": []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "backend"}}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		}

		code, _, err := trigger.HandleWebhook(ctx)
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)

		assert.Zero(t, eventsCtx.Count())
	})

}

func Test__WhitelistedAction__ValidAction(t *testing.T) {
	trigger := &OnIssue{}

	t.Run("valid action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{
				"action": "open",
			},
		}
		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.True(t, result)
	})

	t.Run("invalid action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{
				"action": "update",
			},
		}

		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.False(t, result)
	})

	t.Run("missing action", func(t *testing.T) {
		data := map[string]any{
			"object_attributes": map[string]any{},
		}

		result := trigger.whitelistedAction(log.NewEntry(log.New()), data, []string{"open", "close"})
		assert.False(t, result)
	})

}

func Test__IssueDerivedActions(t *testing.T) {
	t.Run("nil changes", func(t *testing.T) {
		assert.Nil(t, issueDerivedActions(nil))
	})

	t.Run("label added", func(t *testing.T) {
		changes := map[string]any{
			"labels": map[string]any{
				"previous": []any{map[string]any{"id": float64(206), "title": "API"}},
				"current": []any{
					map[string]any{"id": float64(206), "title": "API"},
					map[string]any{"id": float64(205), "title": "Platform"},
				},
			},
		}
		assert.Equal(t, []string{"labeled"}, issueDerivedActions(changes))
	})

	t.Run("label removed", func(t *testing.T) {
		changes := map[string]any{
			"labels": map[string]any{
				"previous": []any{
					map[string]any{"id": float64(206), "title": "API"},
					map[string]any{"id": float64(205), "title": "Platform"},
				},
				"current": []any{map[string]any{"id": float64(205), "title": "Platform"}},
			},
		}
		assert.Equal(t, []string{"unlabeled"}, issueDerivedActions(changes))
	})

	t.Run("assignee added and removed", func(t *testing.T) {
		changes := map[string]any{
			"assignees": map[string]any{
				"previous": []any{map[string]any{"id": float64(1)}},
				"current":  []any{map[string]any{"id": float64(2)}},
			},
		}
		assert.ElementsMatch(t, []string{"assigned", "unassigned"}, issueDerivedActions(changes))
	})

	t.Run("milestoned", func(t *testing.T) {
		changes := map[string]any{
			"milestone_id": map[string]any{"previous": nil, "current": float64(7)},
		}
		assert.Equal(t, []string{"milestoned"}, issueDerivedActions(changes))
	})

	t.Run("demilestoned", func(t *testing.T) {
		changes := map[string]any{
			"milestone_id": map[string]any{"previous": float64(7), "current": nil},
		}
		assert.Equal(t, []string{"demilestoned"}, issueDerivedActions(changes))
	})

	t.Run("milestone swapped", func(t *testing.T) {
		changes := map[string]any{
			"milestone_id": map[string]any{"previous": float64(7), "current": float64(8)},
		}
		assert.Equal(t, []string{"milestoned"}, issueDerivedActions(changes))
	})

	t.Run("locked", func(t *testing.T) {
		changes := map[string]any{
			"discussion_locked": map[string]any{"previous": false, "current": true},
		}
		assert.Equal(t, []string{"locked"}, issueDerivedActions(changes))
	})

	t.Run("unlocked", func(t *testing.T) {
		changes := map[string]any{
			"discussion_locked": map[string]any{"previous": true, "current": false},
		}
		assert.Equal(t, []string{"unlocked"}, issueDerivedActions(changes))
	})

	t.Run("edited title", func(t *testing.T) {
		changes := map[string]any{
			"title": map[string]any{"previous": "old", "current": "new"},
		}
		assert.Equal(t, []string{"edited"}, issueDerivedActions(changes))
	})

	t.Run("edited description", func(t *testing.T) {
		changes := map[string]any{
			"description": map[string]any{"previous": "old", "current": "new"},
		}
		assert.Equal(t, []string{"edited"}, issueDerivedActions(changes))
	})

	t.Run("unrelated change yields no derived actions", func(t *testing.T) {
		changes := map[string]any{
			"updated_at": map[string]any{"previous": "t1", "current": "t2"},
		}
		assert.Empty(t, issueDerivedActions(changes))
	})

	t.Run("no-op lock entry yields no derived actions", func(t *testing.T) {
		changes := map[string]any{
			"discussion_locked": map[string]any{"previous": false, "current": false},
		}
		assert.Empty(t, issueDerivedActions(changes))
	})

	t.Run("no-op milestone entry yields no derived actions", func(t *testing.T) {
		changes := map[string]any{
			"milestone_id": map[string]any{"previous": float64(7), "current": float64(7)},
		}
		assert.Empty(t, issueDerivedActions(changes))
	})
}

func Test__OnIssue__HandleWebhook__DerivedActions(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")
	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}

	t.Run("labeled matches even though update and state are not selected", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": map[string]any{
				"state":  "closed",
				"action": "update",
			},
			"changes": map[string]any{
				"labels": map[string]any{
					"previous": []any{},
					"current":  []any{map[string]any{"id": float64(1), "title": "bug"}},
				},
			},
		}
		body, _ := json.Marshal(data)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"labeled"}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventsCtx.Count())
	})

	t.Run("no match when derived action is not selected", func(t *testing.T) {
		eventsCtx := &contexts.EventContext{}
		data := map[string]any{
			"object_attributes": map[string]any{
				"state":  "opened",
				"action": "update",
			},
			"changes": map[string]any{
				"labels": map[string]any{
					"previous": []any{},
					"current":  []any{map[string]any{"id": float64(1), "title": "bug"}},
				},
			},
		}
		body, _ := json.Marshal(data)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers:       headers,
			Body:          body,
			Configuration: map[string]any{"project": "123", "actions": []string{"assigned"}},
			Webhook:       webhookCtx,
			Events:        eventsCtx,
			Logger:        log.NewEntry(log.New()),
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventsCtx.Count())
	})
}

func Test__OnIssue__HandleWebhook__UpdateOnClosed(t *testing.T) {
	trigger := &OnIssue{}

	headers := http.Header{}
	headers.Set("X-Gitlab-Event", "Issue Hook")
	headers.Set("X-Gitlab-Token", "token")

	webhookCtx := &contexts.NodeWebhookContext{Secret: "token"}
	eventsCtx := &contexts.EventContext{}

	data := map[string]any{
		"object_attributes": map[string]any{
			"state":  "closed",
			"action": "update",
		},
	}
	body, _ := json.Marshal(data)

	ctx := core.WebhookRequestContext{
		Headers:       headers,
		Body:          body,
		Configuration: map[string]any{"project": "123", "actions": []string{"update"}},
		Webhook:       webhookCtx,
		Events:        eventsCtx,
		Logger:        log.NewEntry(log.New()),
	}

	code, _, err := trigger.HandleWebhook(ctx)
	assert.Equal(t, http.StatusOK, code)
	assert.NoError(t, err)

	assert.Equal(t, 0, eventsCtx.Count())
}
