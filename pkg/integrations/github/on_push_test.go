package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPush__HandleWebhook(t *testing.T) {
	trigger := &OnPush{}

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", "push")
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("no X-GitHub-Event -> 400", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Events:  &contexts.EventContext{},
			Webhook: &contexts.NodeWebhookContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-GitHub-Event header")
	})

	t.Run("event is not push -> 200", func(t *testing.T) {
		body := []byte(`{}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "ping")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", "push")

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"ref":"refs/heads/main"}`),
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("branch deletion push is ignored", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main","deleted":true}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("ref is equal -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/main"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref notEquals match -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/feat/1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeNotEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/feat/1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeMatches, Value: "refs/heads/feat/*"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("ref is not equal -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"ref":"refs/heads/patch-1"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", "push")

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})
}

func Test__OnPush__Setup(t *testing.T) {
	helloRepo := Repository{ID: 123456, Name: "hello", URL: "https://github.com/testhq/hello"}
	trigger := OnPush{}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}
		err := trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": "world"},
		})

		require.ErrorContains(t, err, "repository world is not accessible to app installation")
	})

	t.Run("metadata is set and webhook is requested", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Metadata: Metadata{
				Repositories: []Repository{helloRepo},
			},
		}

		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Equal(t, nodeMetadataCtx.Get(), NodeMetadata{Repository: &helloRepo})
		require.Len(t, integrationCtx.WebhookRequests, 1)

		webhookRequest := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		assert.Equal(t, webhookRequest.EventType, "push")
		assert.Equal(t, webhookRequest.Repository, "hello")
	})
}

func Test__IsBranchDeletionEvent(t *testing.T) {
	assert.True(t, isBranchDeletionEvent(map[string]any{"deleted": true}))
	assert.False(t, isBranchDeletionEvent(map[string]any{"deleted": false}))
	assert.False(t, isBranchDeletionEvent(map[string]any{}))
	assert.False(t, isBranchDeletionEvent(map[string]any{}))
}

func Test__OnPush__PathFilter(t *testing.T) {
	trigger := &OnPush{}
	secret := "test-secret"

	makeBody := func(payload string) ([]byte, string) {
		body := []byte(payload)
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		return body, fmt.Sprintf("%x", h.Sum(nil))
	}

	makeHeaders := func(sig string) http.Header {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+sig)
		headers.Set("X-GitHub-Event", "push")
		return headers
	}

	t.Run("no paths configured -> event is always emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":["README.md"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("path glob matches modified file -> event is emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":["pkg/integrations/github/on_push.go"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("path glob matches added file -> event is emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":["pkg/models/new_model.go"],"modified":[],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/models/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("path glob matches removed file -> event is emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":[],"removed":["web_src/src/old.ts"]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"web_src/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("path glob does not match any file -> event is not emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":["README.md","docs/setup.md"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("path glob exact match -> event is emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":["go.sum"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"go.sum"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("single * does not cross path separators -> event is not emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":[],"modified":["pkg/integrations/github/on_push.go"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/integrations/*"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})

	t.Run("path glob matches across multiple commits -> event is emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main","commits":[{"added":["README.md"],"modified":[],"removed":[]},{"added":[],"modified":["pkg/models/canvas.go"],"removed":[]}]}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, eventContext.Count())
	})

	t.Run("path filter configured but payload has no commits -> event is not emitted", func(t *testing.T) {
		body, sig := makeBody(`{"ref":"refs/heads/main"}`)
		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: makeHeaders(sig),
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"refs":       []configuration.Predicate{{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"}},
				"paths":      []string{"pkg/**"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, eventContext.Count())
	})
}

func Test__ExtractChangedFiles(t *testing.T) {
	t.Run("no commits key -> empty slice", func(t *testing.T) {
		assert.Empty(t, extractChangedFiles(map[string]any{}))
	})

	t.Run("empty commits -> empty slice", func(t *testing.T) {
		assert.Empty(t, extractChangedFiles(map[string]any{"commits": []any{}}))
	})

	t.Run("single commit with all categories", func(t *testing.T) {
		data := map[string]any{
			"commits": []any{
				map[string]any{
					"added":    []any{"pkg/new.go"},
					"modified": []any{"pkg/existing.go"},
					"removed":  []any{"pkg/old.go"},
				},
			},
		}
		files := extractChangedFiles(data)
		assert.ElementsMatch(t, []string{"pkg/new.go", "pkg/existing.go", "pkg/old.go"}, files)
	})

	t.Run("multiple commits are aggregated", func(t *testing.T) {
		data := map[string]any{
			"commits": []any{
				map[string]any{
					"added":    []any{"a.go"},
					"modified": []any{},
					"removed":  []any{},
				},
				map[string]any{
					"added":    []any{},
					"modified": []any{"b.go"},
					"removed":  []any{"c.go"},
				},
			},
		}
		files := extractChangedFiles(data)
		assert.ElementsMatch(t, []string{"a.go", "b.go", "c.go"}, files)
	})
}
