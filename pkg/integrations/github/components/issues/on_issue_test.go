package issues

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
	mocks "github.com/superplanehq/superplane/test/support/mocks/github"
)

func Test__OnIssue__HandleWebhook(t *testing.T) {
	trigger := &OnIssue{}
	eventType := "issues"

	t.Run("no X-Hub-Signature-256 -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-GitHub-Event", eventType)
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

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", eventType)

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"action":"opened"}`),
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("action is in list -> event is emitted", func(t *testing.T) {
		body := []byte(`{"action":"opened"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 1)
	})

	t.Run("action is not in list -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"action":"closed"}`)

		secret := "test-secret"
		h := hmac.New(sha256.New, []byte(secret))
		h.Write(body)
		signature := fmt.Sprintf("%x", h.Sum(nil))

		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256="+signature)
		headers.Set("X-GitHub-Event", eventType)

		eventContext := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository": "test",
				"actions":    []string{"opened"},
			},
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, eventContext.Count(), 0)
	})

	t.Run("label filter drops an issue that has none of the labels", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"labels":[{"name":"docs"}]}}`)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":        "acme/widgets",
			"actions":           []string{"opened"},
			"labels":            []string{"bug"},
			"authorsWithAccess": true,
		}, events, httpCtx))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("label filter keeps an issue that has a configured label", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"labels":[{"name":"bug"},{"name":"docs"}]}}`)
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository": "acme/widgets",
			"actions":    []string{"opened"},
			"labels":     []any{"bug"},
		}, events, nil))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.Equal(t, "github.issue", events.Payloads[0].Type)
		assert.Equal(t, map[string]any{
			"action": "opened",
			"issue": map[string]any{
				"labels": []any{
					map[string]any{"name": "bug"},
					map[string]any{"name": "docs"},
				},
			},
		}, events.Payloads[0].Data)
	})

	t.Run("exclude mode drops an issue that has a configured label", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"labels":[{"name":"bug"}]}}`)
		events := &contexts.EventContext{}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":      "acme/widgets",
			"actions":         []string{"opened"},
			"labels":          []string{"bug"},
			"labelFilterMode": "exclude",
		}, events, nil))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("superplane flag keeps other actions and only a labeled open issue", func(t *testing.T) {
		opened := []byte(`{"action":"opened","issue":{"state":"open","labels":[]}}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(signedIssueContext(opened, map[string]any{
			"repository":           "acme/widgets",
			"actions":              []string{"opened", "labeled"},
			"superplaneLabelAdded": true,
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())

		wrongLabel := []byte(`{"action":"labeled","label":{"name":"bug"},"issue":{"state":"open"}}`)
		events = &contexts.EventContext{}
		code, _, err = trigger.HandleWebhook(signedIssueContext(wrongLabel, map[string]any{
			"repository":           "acme/widgets",
			"actions":              []string{"labeled"},
			"superplaneLabelAdded": true,
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())

		closed := []byte(`{"action":"labeled","label":{"name":"superplane"},"issue":{"state":"closed"}}`)
		events = &contexts.EventContext{}
		code, _, err = trigger.HandleWebhook(signedIssueContext(closed, map[string]any{
			"repository":           "acme/widgets",
			"actions":              []string{"labeled"},
			"superplaneLabelAdded": true,
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())

		match := []byte(`{"action":"labeled","label":{"name":"superplane"},"issue":{"state":"open"}}`)
		events = &contexts.EventContext{}
		code, _, err = trigger.HandleWebhook(signedIssueContext(match, map[string]any{
			"repository":           "acme/widgets",
			"actions":              []string{"labeled"},
			"superplaneLabelAdded": true,
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("assignment filter uses issue assignees", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"assignees":[{"login":"octocat"}]}}`)
		events := &contexts.EventContext{}
		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository": "acme/widgets",
			"actions":    []string{"opened"},
			"assignment": "unassigned",
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())

		events = &contexts.EventContext{}
		code, _, err = trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository": "acme/widgets",
			"actions":    []string{"opened"},
			"assignment": "assigned",
		}, events, nil))
		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 1, events.Count())
	})

	t.Run("collaborator filter calls GitHub once and emits the webhook body", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"user":{"login":"octocat"},"labels":[{"name":"bug"}]}}`)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{"permission":"write","role_name":"write"}`),
			},
		}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":        "acme/widgets",
			"actions":           []string{"opened"},
			"labels":            []string{"bug"},
			"authorsWithAccess": true,
		}, events, httpCtx))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, events.Count())
		assert.NotContains(t, events.Payloads[0].Data, "permission")
		require.Len(t, httpCtx.Requests, 1)
		assert.Contains(t, httpCtx.Requests[0].URL.Path, "/repos/acme/widgets/collaborators/octocat/permission")
	})

	t.Run("collaborator filter does not emit when permission is none", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"user":{"login":"octocat"}}}`)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{"permission":"none","role_name":"none"}`),
			},
		}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":        "acme/widgets",
			"actions":           []string{"opened"},
			"authorsWithAccess": true,
		}, events, httpCtx))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Len(t, httpCtx.Requests, 1)
	})

	t.Run("missing author login does not call GitHub", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{}}`)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":        "acme/widgets",
			"actions":           []string{"opened"},
			"authorsWithAccess": true,
		}, events, httpCtx))

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Equal(t, 0, events.Count())
		assert.Empty(t, httpCtx.Requests)
	})

	t.Run("permission API error does not emit and returns 500", func(t *testing.T) {
		body := []byte(`{"action":"opened","issue":{"user":{"login":"octocat"}}}`)
		events := &contexts.EventContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusInternalServerError, `{"message":"boom"}`),
			},
		}

		code, _, err := trigger.HandleWebhook(signedIssueContext(body, map[string]any{
			"repository":        "acme/widgets",
			"actions":           []string{"opened"},
			"authorsWithAccess": true,
		}, events, httpCtx))

		assert.Equal(t, http.StatusInternalServerError, code)
		assert.Error(t, err)
		assert.Equal(t, 0, events.Count())
	})

	t.Run("invalid signature does not call GitHub", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Hub-Signature-256", "sha256=asdasd")
		headers.Set("X-GitHub-Event", eventType)
		httpCtx := &contexts.HTTPContext{}

		code, _, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"action":"opened","issue":{"user":{"login":"octocat"}}}`),
			Headers: headers,
			Logger:  logrus.NewEntry(logrus.New()),
			Configuration: map[string]any{
				"repository":        "acme/widgets",
				"actions":           []string{"opened"},
				"authorsWithAccess": true,
			},
			Webhook:     &contexts.NodeWebhookContext{Secret: "test-secret"},
			Events:      &contexts.EventContext{},
			HTTP:        httpCtx,
			Integration: mocks.IntegrationContextForNewSetupFlow(),
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
		assert.Empty(t, httpCtx.Requests)
	})
}

func signedIssueContext(
	body []byte,
	configuration map[string]any,
	events *contexts.EventContext,
	httpCtx core.HTTPContext,
) core.WebhookRequestContext {
	secret := "test-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	headers := http.Header{}
	headers.Set("X-Hub-Signature-256", "sha256="+fmt.Sprintf("%x", mac.Sum(nil)))
	headers.Set("X-GitHub-Event", "issues")
	return core.WebhookRequestContext{
		Body:          body,
		Headers:       headers,
		Logger:        logrus.NewEntry(logrus.New()),
		Configuration: configuration,
		Webhook:       &contexts.NodeWebhookContext{Secret: secret},
		Events:        events,
		HTTP:          httpCtx,
		Integration:   mocks.IntegrationContextForNewSetupFlow(),
	}
}

func Test__OnIssue__Setup(t *testing.T) {
	trigger := OnIssue{}

	t.Run("webhook is requested", func(t *testing.T) {
		integrationCtx := mocks.IntegrationContextForNewSetupFlow()
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				mocks.GitHubResponse(http.StatusOK, `{
					"id": 123456,
					"name": "hello",
					"html_url": "https://github.com/testhq/hello"
				}`),
			},
		}
		nodeMetadataCtx := contexts.MetadataContext{}
		require.NoError(t, trigger.Setup(core.TriggerContext{
			Integration:   integrationCtx,
			HTTP:          httpCtx,
			Metadata:      &nodeMetadataCtx,
			Configuration: map[string]any{"repository": "hello"},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest := integrationCtx.WebhookRequests[0].(common.WebhookConfiguration)
		assert.Equal(t, webhookRequest.EventType, "issues")
		assert.Equal(t, webhookRequest.Repository, "hello")
	})
}
