package bitbucket

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
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	contexts "github.com/superplanehq/superplane/test/support/contexts"
)

func Test__OnPush__Setup(t *testing.T) {
	trigger := OnPush{}
	metadata := Metadata{
		AuthType: AuthTypeWorkspaceAccessToken,
		Workspace: &WorkspaceMetadata{
			Slug: "superplane",
		},
	}

	t.Run("repository is required", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:          &contexts.HTTPContext{},
			Integration:   integrationCtx,
			Metadata:      &contexts.MetadataContext{},
			Configuration: map[string]any{"repository": ""},
		})

		require.ErrorContains(t, err, "repository is required")
	})

	t.Run("repository is not accessible", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"uuid":"{hello}","name":"hello","full_name":"superplane/hello","slug":"hello"}]}`,
					)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}

		err := trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
			Configuration: map[string]any{
				"repository": "world",
			},
		})

		require.ErrorContains(t, err, "repository world is not accessible to workspace")
	})

	t.Run("metadata is set and webhook is requested", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(
						`{"values":[{"uuid":"{hello}","name":"hello","full_name":"superplane/hello","slug":"hello"}]}`,
					)),
				},
			},
		}
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"token": "token"},
			Metadata:      metadata,
		}
		nodeMetadataCtx := &contexts.MetadataContext{}

		require.NoError(t, trigger.Setup(core.TriggerContext{
			HTTP:        httpCtx,
			Integration: integrationCtx,
			Metadata:    nodeMetadataCtx,
			Configuration: map[string]any{
				"repository": "hello",
			},
		}))

		require.Len(t, integrationCtx.WebhookRequests, 1)
		webhookRequest, ok := integrationCtx.WebhookRequests[0].(WebhookConfiguration)
		require.True(t, ok)
		assert.Equal(t, []string{"repo:push"}, webhookRequest.EventTypes)
		assert.Equal(t, "hello", webhookRequest.RepositorySlug)

		require.NotNil(t, nodeMetadataCtx.Metadata)
		nodeMetadata, ok := nodeMetadataCtx.Metadata.(NodeMetadata)
		require.True(t, ok)
		require.NotNil(t, nodeMetadata.Repository)
		assert.Equal(t, "{hello}", nodeMetadata.Repository.UUID)
		assert.Equal(t, "hello", nodeMetadata.Repository.Name)
		assert.Equal(t, "superplane/hello", nodeMetadata.Repository.FullName)
		assert.Equal(t, "hello", nodeMetadata.Repository.Slug)
	})
}

func Test__OnPush__HandleWebhook(t *testing.T) {
	trigger := &OnPush{}

	t.Run("no X-Event-Key -> 400", func(t *testing.T) {
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "missing X-Event-Key header")
	})

	t.Run("event is not repo:push -> 200", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:fork")

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
			Events:  eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("missing signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "missing X-Hub-Signature header")
	})

	t.Run("invalid signature format -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256=")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Headers: headers,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature format")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256=invalid")

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{}`),
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid body -> 400", func(t *testing.T) {
		body := []byte("{")
		signature := signBitbucketPayload("test-secret", body)

		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signature)

		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository": "hello",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Events: &contexts.EventContext{},
		})

		assert.Equal(t, http.StatusBadRequest, code)
		assert.ErrorContains(t, err, "error parsing request body")
	})

	t.Run("ref does not match -> event is not emitted", func(t *testing.T) {
		body := []byte(`{"push":{"changes":[{"new":{"type":"branch","name":"feature-1"}}]}}`)
		signature := signBitbucketPayload("test-secret", body)

		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository": "hello",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		assert.Zero(t, eventContext.Count())
	})

	t.Run("ref matches -> event is emitted", func(t *testing.T) {
		body := []byte(`{"push":{"changes":[{"new":{"type":"branch","name":"main"}}]}}`)
		signature := signBitbucketPayload("test-secret", body)

		headers := http.Header{}
		headers.Set("X-Event-Key", "repo:push")
		headers.Set("X-Hub-Signature", "sha256="+signature)

		eventContext := &contexts.EventContext{}
		code, err := trigger.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.WebhookContext{Secret: "test-secret"},
			Configuration: map[string]any{
				"repository": "hello",
				"refs": []configuration.Predicate{
					{Type: configuration.PredicateTypeEquals, Value: "refs/heads/main"},
				},
			},
			Events: eventContext,
		})

		assert.Equal(t, http.StatusOK, code)
		assert.NoError(t, err)
		require.Equal(t, 1, eventContext.Count())
		assert.Equal(t, "bitbucket.push", eventContext.Payloads[0].Type)
	})
}

func Test__ExtractRef(t *testing.T) {
	t.Run("extracts branch ref", func(t *testing.T) {
		ref := extractRef(map[string]any{
			"push": map[string]any{
				"changes": []any{
					map[string]any{
						"new": map[string]any{
							"type": "branch",
							"name": "main",
						},
					},
				},
			},
		})

		assert.Equal(t, "refs/heads/main", ref)
	})

	t.Run("extracts tag ref", func(t *testing.T) {
		ref := extractRef(map[string]any{
			"push": map[string]any{
				"changes": []any{
					map[string]any{
						"new": map[string]any{
							"type": "tag",
							"name": "v1.2.3",
						},
					},
				},
			},
		})

		assert.Equal(t, "refs/tags/v1.2.3", ref)
	})

	t.Run("missing ref returns empty string", func(t *testing.T) {
		ref := extractRef(map[string]any{
			"push": map[string]any{
				"changes": []any{
					map[string]any{
						"new": map[string]any{
							"type": "branch",
						},
					},
				},
			},
		})

		assert.Empty(t, ref)
	})
}

func signBitbucketPayload(secret string, body []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	return fmt.Sprintf("%x", h.Sum(nil))
}
